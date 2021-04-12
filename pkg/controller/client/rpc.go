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

package client

import (
	"fmt"
	"net/url"

	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/util/logs"
	"github.com/valyala/fasthttp"
)

type ControllerClient struct {
	client  *fasthttp.HostClient
	scheme  string
	host    string
	version string
}

func NewControllerClient(runOptions *ControllerClientOptions) (c *ControllerClient, err error) {
	u, err := url.Parse(runOptions.Host)
	if err != nil {
		return nil, InvaildConfigError{Reason: fmt.Sprintf("invaild host %s", runOptions.Host)}
	}
	client := fasthttp.HostClient{Addr: u.Host}
	return &ControllerClient{
		client:  &client,
		scheme:  u.Scheme,
		version: "v1",
		host:    u.Host,
	}, nil
}

func (c *ControllerClient) Invoke(ir *api.InvokeRequest) (response *api.InvokeResponse, err error) {
	response = api.NewInvokeResponse()
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	p := fmt.Sprintf("%s://%s/%s/functions/%s/invocations", c.scheme, c.host, c.version, ir.FunctionBRN)
	req.SetRequestURI(p)

	req.Header.SetHost(c.host)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Set(api.HeaderXRequestID, ir.RequestID)
	req.Header.Set(api.HeaderXAccountID, ir.UserID)
	if ir.Body != nil {
		req.SetBodyString(*ir.Body)
	}

	err = c.client.Do(req, resp)
	if err != nil {
		logs.Errorf("unexpected controller client error: %s", err)
		// TODO: handler unexpected error
		return
	}
	response.SetStatusCode(resp.StatusCode())
	response.SetBody(resp.Body())
	resp.Header.VisitAll(func(key, value []byte) {
		response.SetHeader(string(key), string(value))
	})
	return
}
