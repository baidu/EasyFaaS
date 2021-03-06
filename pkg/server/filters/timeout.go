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

package filters

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	kunErr "github.com/baidu/easyfaas/pkg/error"
	"github.com/baidu/easyfaas/pkg/util/json"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

var errConnKilled = fmt.Errorf("kill connection/stream")

type timeoutFuncType func(req *http.Request) (*time.Timer, func(), *kunErr.FinalError)

// WithTimeoutFilter times out non-long-running requests after the time given by timeout.
func WithTimeoutFilter(handler http.Handler, timeout time.Duration) http.Handler {
	if timeout == 0 {
		return handler
	}
	timeoutFunc := timeoutFuncType(func(req *http.Request) (*time.Timer, func(), *kunErr.FinalError) {
		reqtimeout := req.URL.Query().Get("timeout")
		if len(reqtimeout) > 0 {
			if numtimeout, err := time.ParseDuration(reqtimeout); err == nil {
				timeout = numtimeout
			} else {
				logs.Errorf("WithTimeoutFilter parse duration err(%+v)", err)
			}
		}
		metricFn := func() {
			// metrics.Record(req, requestInfo, "", http.StatusGatewayTimeout, 0, 0)
		}
		err := kunErr.NewRequestTimeoutException("timeout", timeout, nil)
		return time.NewTimer(timeout), metricFn, &err
	})
	return WithTimeout(handler, timeoutFunc)
}

// WithTimeout returns an http.Handler that runs h with a timeout
// determined by timeoutFunc. The new http.Handler calls h.ServeHTTP to handle
// each request, but if a call runs for longer than its time limit, the
// handler responds with a 504 Gateway Timeout error and the message
// provided. (If msg is empty, a suitable default message will be sent.) After
// the handler times out, writes by h to its http.ResponseWriter will return
// http.ErrHandlerTimeout. If timeoutFunc returns a nil timeout channel, no
// timeout will be enforced. recordFn is a function that will be invoked whenever
// a timeout happens.
func WithTimeout(h http.Handler, timeoutFunc timeoutFuncType) http.Handler {
	return &timeoutHandler{h, timeoutFunc}
}

type timeoutHandler struct {
	handler http.Handler
	timeout timeoutFuncType
}

func (t *timeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	after, recordFn, err := t.timeout(r)
	if after == nil {
		t.handler.ServeHTTP(w, r)
		return
	}

	done := make(chan struct{})
	tw := newTimeoutWriter(w)
	go func() {
		t.handler.ServeHTTP(tw, r)
		close(done)
	}()
	select {
	case <-done:
	case <-after.C:
		recordFn()
		tw.timeout(err)
	}
	after.Stop()
}

type timeoutWriter interface {
	http.ResponseWriter
	timeout(error)
}

func newTimeoutWriter(w http.ResponseWriter) timeoutWriter {
	base := &baseTimeoutWriter{w: w}
	return base
}

type baseTimeoutWriter struct {
	w http.ResponseWriter

	mu sync.Mutex
	// if the timeout handler has timeout
	timedOut bool
	// if this timeout writer has wrote header
	wroteHeader bool
	// if this timeout writer has been hijacked
	hijacked bool
}

func (tw *baseTimeoutWriter) Header() http.Header {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return http.Header{}
	}

	return tw.w.Header()
}

func (tw *baseTimeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	if tw.hijacked {
		return 0, http.ErrHijacked
	}

	tw.wroteHeader = true
	return tw.w.Write(p)
}

func (tw *baseTimeoutWriter) Flush() {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return
	}

	if flusher, ok := tw.w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (tw *baseTimeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut || tw.wroteHeader || tw.hijacked {
		return
	}

	tw.wroteHeader = true
	tw.w.WriteHeader(code)
}

func (tw *baseTimeoutWriter) timeout(err error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	tw.timedOut = true

	// The timeout writer has not been used by the inner handler.
	// We can safely timeout the HTTP request by sending by a timeout
	// handler
	if !tw.wroteHeader && !tw.hijacked {
		tw.w.WriteHeader(http.StatusGatewayTimeout)
		enc := json.NewEncoder(tw.w)
		enc.Encode(err)
	} else {
		// The timeout writer has been used by the inner handler. There is
		// no way to timeout the HTTP request at the point. We have to shutdown
		// the connection for HTTP1 or reset stream for HTTP2.
		//
		// Note from: Brad Fitzpatrick
		// if the ServeHTTP goroutine panics, that will do the best possible thing for both
		// HTTP/1 and HTTP/2. In HTTP/1, assuming you're replying with at least HTTP/1.1 and
		// you've already flushed the headers so it's using HTTP chunking, it'll kill the TCP
		// connection immediately without a proper 0-byte EOF chunk, so the peer will recognize
		// the response as bogus. In HTTP/2 the server will just RST_STREAM the stream, leaving
		// the TCP connection open, but resetting the stream to the peer so it'll have an error,
		// like the HTTP/1 case.
		panic(errConnKilled)
	}
}
