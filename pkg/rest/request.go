/*
 * Copyright (c) 2020 Baidu, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rest

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kunErr "github.com/baidu/easyfaas/pkg/error"
	myjson "github.com/baidu/easyfaas/pkg/rest/serializer/json"
	"github.com/baidu/easyfaas/pkg/util/json"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

// HTTPClient is an interface for testing a request object.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ResponseWrapper is an interface for getting a response.
// The response may be either accessed as a raw data (the whole output is put into memory) or as a stream.
type ResponseWrapper interface {
	DoRaw() ([]byte, error)
	Stream() (io.ReadCloser, error)
}

// Request allows for building up a request to a server in a chained fashion.
// Any errors are stored until the end of your call, so you only have to
// check once.
type Request struct {
	// required
	client HTTPClient
	verb   string

	baseURL *url.URL
	content ContentConfig

	// generic components accessible via method setters
	pathPrefix string
	subpath    string
	params     url.Values
	headers    http.Header
	host       string
	version    string

	// structural elements of the request that are part of the Kubernetes API conventions
	namespace    string
	namespaceSet bool
	resource     string
	resourceName string
	subresource  string
	timeout      time.Duration

	// output
	err  error
	body io.Reader

	// This is only used for per-request timeouts, deadlines, and cancellations.
	ctx context.Context

	backoffMgr BackoffManager
	// throttle   flowcontrol.RateLimiter

	clientTimeout time.Duration
}

const (
	BackendTypeInternal  = "internal"
	BackendTypeK8s       = "k8s"
	BackendTypeBce       = "cloud"
	BackendTypeOpenstack = "openstack"

	HeaderKeyConnection            = "Connection"
	HeaderValueConnectionClose     = "close"
	HeaderValueConnectionKeepAlive = "Keep-Alive"

	HeaderKeyKeepAlive          = "Keep-Alive"
	HeaderValueKeepAliveDefault = "timeout=5, max=100"
)

// NewRequest creates a new request helper object for accessing runtime.Objects on a server.
// TODO: add client-side throttle & backoff like k8s
// TODO: more serializers
func NewRequest(client HTTPClient, verb string, baseURL *url.URL, version string, content ContentConfig, backoff BackoffManager, timeout time.Duration) *Request {
	if backoff == nil {
		logs.Debugf("Not implementing request backoff strategy")
		backoff = &NoBackoff{}
	}

	pathPrefix := "/"
	if baseURL != nil {
		pathPrefix = path.Join(pathPrefix, baseURL.Path)
	}
	r := &Request{
		client:        client,
		verb:          verb,
		baseURL:       baseURL,
		content:       content,
		pathPrefix:    path.Join(pathPrefix, version),
		version:       version,
		backoffMgr:    backoff,
		timeout:       timeout,
		clientTimeout: content.ClientTimeout,
	}
	switch {
	case len(content.AcceptContentTypes) > 0:
		r.SetHeader("Accept", content.AcceptContentTypes)
	case len(content.ContentType) > 0:
		r.SetHeader("Accept", content.ContentType+", */*")
	}

	if content.Connection != "" {
		r.SetHeader(HeaderKeyConnection, content.Connection)

		if content.Connection == HeaderValueConnectionKeepAlive {
			if content.KeepAlive == "" {
				r.SetHeader(HeaderKeyKeepAlive, HeaderValueKeepAliveDefault)
			} else {
				r.SetHeader(HeaderKeyKeepAlive, content.KeepAlive)
			}
		}
	}

	// Add simple auth token between internal rpc service
	if content.BackendType == BackendTypeInternal {
		r.SetHeader("X-Auth-Token", defaultInternalAuthToken)
	}

	return r
}

func (r *Request) SetHost(host string) *Request {
	r.host = host
	return r
}

func (r *Request) Host() string {
	return r.host
}

func (r *Request) SetHeader(key string, values ...string) *Request {
	if r.headers == nil {
		r.headers = http.Header{}
	}
	r.headers.Del(key)
	for _, value := range values {
		r.headers.Add(key, value)
	}
	return r
}

func (r *Request) Header() http.Header {
	return r.headers
}

func (r *Request) Verb() string {
	return strings.ToUpper(r.verb)
}

// Timeout makes the request use the given duration as a timeout. Sets the "timeout"
// parameter.
func (r *Request) Timeout(d time.Duration) *Request {
	if r.err != nil {
		return r
	}
	r.timeout = d
	return r
}

// ClientTimeout
func (r *Request) ClientTimeout(d time.Duration) *Request {
	r.clientTimeout = d
	return r
}

func (r *Request) BaseURL(baseURL *url.URL) *Request {
	r.baseURL = baseURL
	return r
}

// Resource sets the resource to access (<resource>/[ns/<namespace>/]<name>)
func (r *Request) Resource(resource string) *Request {
	if r.err != nil {
		return r
	}
	if len(r.resource) != 0 {
		r.err = fmt.Errorf("resource already set to %q, cannot change to %q", r.resource, resource)
		return r
	}
	// if msgs := IsValidPathSegmentName(resource); len(msgs) != 0 {
	// 	r.err = fmt.Errorf("invalid resource %q: %v", resource, msgs)
	// 	return r
	// }
	r.resource = resource
	return r
}

// Namespace applies the namespace scope to a request (<resource>/[ns/<namespace>/]<name>)
func (r *Request) Namespace(namespace string) *Request {
	if r.err != nil {
		return r
	}
	if r.namespaceSet {
		r.err = fmt.Errorf("namespace already set to %q, cannot change to %q", r.namespace, namespace)
		return r
	}
	// if msgs := IsValidPathSegmentName(namespace); len(msgs) != 0 {
	// 	r.err = fmt.Errorf("invalid namespace %q: %v", namespace, msgs)
	// 	return r
	// }
	r.namespaceSet = true
	r.namespace = namespace
	return r
}

// Body makes the request use obj as the body. Optional.
// If obj is a string, try to read a file of that name.
// If obj is a []byte, send it directly.
// If obj is an io.Reader, use it directly.
// If obj is a object, marshal it with json, and set Content-Type header.
// TODO: more serializers
func (r *Request) Body(obj interface{}) *Request {
	if r.err != nil {
		return r
	}
	switch t := obj.(type) {
	case string:
		data, err := ioutil.ReadFile(t)
		if err != nil {
			r.err = err
			return r
		}
		glogBody("Request Body", data)
		r.body = bytes.NewReader(data)
	case []byte:
		glogBody("Request Body", t)
		r.body = bytes.NewReader(t)
	case io.Reader:
		r.body = t
	default:
		data, err := json.Marshal(obj)
		if err != nil {
			r.err = err
			return r
		}
		glogBody("Request Body", data)
		r.body = bytes.NewReader(data)
		if len(r.headers.Get("Content-Type")) == 0 {
			r.SetHeader("Content-Type", "application/json")
		}
	}
	return r
}

// Context adds a context to the request. Contexts are only used for
// timeouts, deadlines, and cancellations.
func (r *Request) Context(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// URL returns the current working URL.
func (r *Request) URL() *url.URL {
	p := r.pathPrefix
	if r.namespaceSet && len(r.namespace) > 0 {
		p = path.Join(p, "namespaces", r.namespace)
	}
	if len(r.resource) != 0 {
		p = path.Join(p, r.resource)
	}
	finalURL := &url.URL{}
	if r.baseURL != nil {
		*finalURL = *r.baseURL
	}
	finalURL.Path = p

	query := url.Values{}
	for key, values := range r.params {
		for _, value := range values {
			query.Add(key, value)
		}
	}

	// timeout is handled specially here.
	if r.timeout != 0 {
		query.Set("timeout", r.timeout.String())
	}
	finalURL.RawQuery = query.Encode()
	return finalURL
}

// Param creates a query parameter with the given string value.
func (r *Request) Param(paramName, s string) *Request {
	if r.err != nil {
		return r
	}
	return r.setParam(paramName, s)
}

func (r *Request) setParam(paramName, value string) *Request {
	if r.params == nil {
		r.params = make(url.Values)
	}
	r.params[paramName] = append(r.params[paramName], value)
	return r
}

func (r *Request) GetParams() url.Values {
	return r.params
}

// Criteria adds special parameters into this request
func (r *Request) Criteria(criteria QueryCriteria) *Request {
	if criteria == nil {
		return r
	}

	if r.params == nil {
		r.params = criteria.Value()
	} else {
		for key, values := range criteria.Value() {
			for _, value := range values {
				r.Param(key, value)
			}
		}
	}

	return r
}

// request connects to the server and invokes the provided function when a server response is
// received. It handles retry behavior and up front validation of requests. It will invoke
// fn at most once. It will return an error if a problem occurred prior to connecting to the
// server - the provided function is responsible for handling server errors.
func (r *Request) request(fn func(*http.Request, *http.Response)) error {
	// Metrics for total request latency
	start := time.Now()
	defer func() {
		// TODO: log cost into observer
		// metrics.RequestLatency.Observe(r.verb, r.finalURLTemplate(), time.Since(start))
		logs.Debugf("%s %s, cost %s", r.Verb(), r.URL(), time.Since(start))
	}()

	if r.err != nil {
		logs.Warnf("Error in request: %v", r.err)
		return r.err
	}

	client := r.client
	if client == nil {
		client = http.DefaultClient
	}

	if r.clientTimeout == 0 {
		r.clientTimeout = 60 * time.Second
	}

	if r.clientTimeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), r.clientTimeout)
		defer cancel()
		r.ctx = ctx
	}

	// Right now we make about ten retry attempts if we get a Retry-After response.
	// TODO: Change to a timeout based approach.
	maxRetries := 10
	retries := 0
	for {
		url := r.URL().String()
		req, err := http.NewRequest(r.verb, url, r.body)
		if err != nil {
			return err
		}
		if r.ctx != nil {
			req = req.WithContext(r.ctx)
		}
		req.Header = r.headers

		if len(r.host) != 0 {
			req.Host = r.host
		}

		r.backoffMgr.Sleep(r.backoffMgr.CalculateBackoff(r.URL()))
		resp, err := client.Do(req)
		if err != nil {
			r.backoffMgr.UpdateBackoff(r.URL(), err, 0)
		} else {
			r.backoffMgr.UpdateBackoff(r.URL(), err, resp.StatusCode)
		}
		if err != nil {
			// "Connection reset by peer" is usually a transient error.
			// Thus in case of "GET" operations, we simply retry it.
			// We are not automatically retrying "write" operations, as
			// they are not idempotent.
			if !isConnectionReset(err) || r.verb != "GET" {
				return err
			}
			// For the purpose of retry, we set the artificial "retry-after" response.
			// TODO: Should we clean the original response if it exists?
			resp = &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     http.Header{"Retry-After": []string{"1"}},
				Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			}
		}

		done := func() bool {
			// Ensure the response body is fully read and closed
			// before we reconnect, so that we reuse the same TCP
			// connection.
			defer func() {
				const maxBodySlurpSize = 2 << 10
				if resp.ContentLength <= maxBodySlurpSize {
					io.Copy(ioutil.Discard, &io.LimitedReader{R: resp.Body, N: maxBodySlurpSize})
				}
				resp.Body.Close()
			}()

			retries++
			if seconds, wait := checkWait(resp); wait && retries < maxRetries {
				if seeker, ok := r.body.(io.Seeker); ok && r.body != nil {
					_, err := seeker.Seek(0, 0)
					if err != nil {
						logs.Warnf("Could not retry request, can't Seek() back to beginning of body for %T", r.body)
						fn(req, resp)
						return true
					}
				}

				logs.Debugf("Got a Retry-After %d response for attempt %d to %v", seconds, retries, url)
				r.backoffMgr.Sleep(time.Duration(seconds) * time.Second)
				return false
			}
			fn(req, resp)
			return true
		}()
		if done {
			return nil
		}
	}
}

// Do formats and executes the request. Returns a Result object for easy response
// processing.
//
// Error type:
//  * If the request can't be constructed, or an error happened earlier while building its
//    arguments: *RequestConstructionError
//  * If the server responds with a status: *errors.StatusError or *errors.UnexpectedObjectError
//  * http.Client.Do errors are returned directly.
func (r *Request) Do() Result {
	var result Result
	err := r.request(func(req *http.Request, resp *http.Response) {
		result = r.transformResponse(resp, req)
	})
	if err != nil {
		return Result{err: err}
	}
	return result
}

// transformResponse converts an API response into a structured API object
func (r *Request) transformResponse(resp *http.Response, req *http.Request) Result {
	var body []byte
	if resp.Body != nil {
		if data, err := ioutil.ReadAll(resp.Body); err == nil {
			body = data
		}
	}

	glogBody("Response Body", body)

	// verify the content type is accurate
	contentType := resp.Header.Get("Content-Type")
	decoder := myjson.NewSerializer()

	switch {
	case resp.StatusCode == http.StatusSwitchingProtocols:
		// no-op, we've been upgraded
	case resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusPartialContent:
		// calculate an unstructured error from the response which the Result object may use if the caller
		// did not return a structured error.
		// retryAfter, _ := retryAfterSeconds(resp)
		err := errorFromBody(r.content.BackendType, decoder, body)
		return Result{
			body:        body,
			header:      resp.Header,
			contentType: contentType,
			statusCode:  resp.StatusCode,
			decoder:     decoder,
			backend:     r.content.BackendType,
			err:         err,
		}
	}

	return Result{
		body:        body,
		header:      resp.Header,
		contentType: contentType,
		statusCode:  resp.StatusCode,
		decoder:     decoder,
		backend:     r.content.BackendType,
	}
}

// glogBody logs a body output that could be either JSON or protobuf. It explicitly guards against
// allocating a new string for the body output unless necessary. Uses a simple heuristic to determine
// whether the body is printable.
func glogBody(prefix string, body []byte) {
	if logs.Check(logs.V(8)) {
		if bytes.IndexFunc(body, func(r rune) bool {
			return r < 0x0a
		}) != -1 {
			logs.Infof("%s:\n%s", prefix, hex.Dump(body))
		} else {
			logs.Infof("%s: %s", prefix, string(body))
		}
	}
}

// checkWait returns true along with a number of seconds if the server instructed us to wait
// before retrying.
func checkWait(resp *http.Response) (int, bool) {
	switch r := resp.StatusCode; {
	// any 500 error code and 429 can trigger a wait
	case r == http.StatusTooManyRequests, r >= 500:
	default:
		return 0, false
	}
	i, ok := retryAfterSeconds(resp)
	return i, ok
}

// retryAfterSeconds returns the value of the Retry-After header and true, or 0 and false if
// the header was missing or not a valid number.
func retryAfterSeconds(resp *http.Response) (int, bool) {
	if h := resp.Header.Get("Retry-After"); len(h) > 0 {
		if i, err := strconv.Atoi(h); err == nil {
			return i, true
		}
	}
	return 0, false
}

// Result contains the result of calling Request.Do().
type Result struct {
	body        []byte
	contentType string
	err         error
	statusCode  int
	backend     string
	header      http.Header
	decoder     Decoder
}

// Raw returns the raw result.
func (r Result) Raw() ([]byte, error) {
	return r.body, r.err
}

// Header returns HTTP Header
func (r Result) Header() http.Header {
	return r.header
}

// StatusCode returns the HTTP status code of the request. (Only valid if no
// error was returned.)
func (r Result) StatusCode(statusCode *int) Result {
	*statusCode = r.statusCode
	return r
}

func (r Result) GetStatusCode() int {
	return r.statusCode
}

// Into stores the result into obj, if possible. If obj is nil it is ignored.
// If the returned object is of type Status and has .Status != StatusSuccess, the
// additional information in Status will be used to enrich the error.
func (r Result) Into(obj interface{}) error {
	if r.err != nil {
		// Check whether the result has a Status object in the body and prefer that.
		return r.Error()
	}
	if r.decoder == nil {
		return fmt.Errorf("serializer for %s doesn't exist", r.contentType)
	}
	if len(r.body) == 0 {
		return fmt.Errorf("0-length response")
	}

	return r.decoder.Decode(r.body, obj)
}

// Error returns the error executing the request, nil if no error occurred.
// If the returned object is of type Status and has Status != StatusSuccess, the
// additional information in Status will be used to enrich the error.
// See the Request.Do() comment for what errors you might get.
func (r Result) Error() error {
	// if we have received an unexpected server error, and we have a body and decoder, we can try to extract
	// a Status object.
	if r.err == nil || len(r.body) == 0 || r.decoder == nil {
		return r.err
	}
	return r.errorFromBody()
}

// SetError for UT purpose
func (r *Result) SetError(err error) {
	r.err = err
}

// StatusError xxx
type StatusError struct {
	ErrStatus metav1.Status
}

// Error implements the Error interface.
func (e StatusError) Error() string {
	return e.ErrStatus.Message
}

// BceError xxx
type BceError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

func (e BceError) Error() string {
	return e.Message
}

// NeutronErrorDetail xxx
type NeutronErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Detail  string `json:"detail"`
}

// NeutronError xxx
type NeutronError struct {
	NeutronError NeutronErrorDetail `json:"NeutronError"`
}

// Error xxx
func (e NeutronError) Error() string {
	return e.NeutronError.Message
}

func (r Result) errorFromBody() error {
	return errorFromBody(r.backend, r.decoder, r.body)
}

func errorFromBody(backend string, decoder Decoder, body []byte) (err error) {
	switch backend {
	case BackendTypeInternal:
		myerr := kunErr.FinalError{}
		if err = decoder.Decode(body, &myerr); err == nil {
			return myerr
		}
	case BackendTypeK8s:
		status := metav1.Status{}
		if err = decoder.Decode(body, &status); err == nil {
			return StatusError{ErrStatus: status}
		}
	case BackendTypeBce:
		myerr := BceError{}
		if err = decoder.Decode(body, &myerr); err == nil {
			return myerr
		}
	case BackendTypeOpenstack:
		myerr := NeutronError{}
		if err = decoder.Decode(body, &myerr); err == nil {
			return myerr
		}
	default:
		logs.Errorf("unknown error type=%s response=%s", backend, string(body))
		return errors.New(string(body))
	}
	if err != nil {
		logs.Errorf("unknown error msg=%s", err.Error())
	}
	return err
}

// Returns if the given err is "connection reset by peer" error.
func isConnectionReset(err error) bool {
	opErr, ok := err.(*net.OpError)
	if ok && opErr.Err.Error() == syscall.ECONNRESET.Error() {
		return true
	}
	return false
}
