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
	"errors"
	"net/http"

	"github.com/valyala/fasthttp"
)

type FastClient struct {
	client *fasthttp.HostClient
}

// FastHTTPClient is an interface for testing a request object.
type FastHTTPClient interface {
	Do(req *fasthttp.Request) (resp *fasthttp.Response, err error)
}

func NewFastClient(host string) *FastClient {
	return &FastClient{client: &fasthttp.HostClient{Addr: host}}
}

func (c *FastClient) Do(req *fasthttp.Request) (resp *fasthttp.Response, err error) {
	resp = fasthttp.AcquireResponse()
	if err := c.client.Do(req, resp); err != nil {
		return resp, err
	}
	switch {
	case resp.StatusCode() == http.StatusSwitchingProtocols:
		// no-op, we've been upgraded
	case resp.StatusCode() < http.StatusOK || resp.StatusCode() > http.StatusPartialContent:
		return resp, errors.New(string(resp.Body()))
	}
	return resp, err
}
