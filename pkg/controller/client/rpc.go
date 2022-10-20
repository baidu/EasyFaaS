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
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/baidu/easyfaas/pkg/rest"
	"github.com/baidu/easyfaas/pkg/controller/rtctrl"
	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/util/logs"
	"github.com/valyala/fasthttp"
)

type ControllerClient struct {
	client  rest.FastHTTPClient
	scheme  string
	host    string
	prefix  string
	version string
}

func NewControllerClient(runOptions *ControllerClientOptions) (c *ControllerClient, err error) {
	u, err := url.Parse(runOptions.Host)
	if err != nil {
		return nil, InvaildConfigError{Reason: fmt.Sprintf("invaild host %s", runOptions.Host)}
	}
	client := rest.NewFastClient(u.Host)
	return &ControllerClient{
		client:  client,
		scheme:  u.Scheme,
		prefix:  u.Path,
		version: "v1",
		host:    u.Host,
	}, nil
}

func (c *ControllerClient) Invoke(ir *api.InvokeRequest) (response *api.InvokeResponse, err error) {
	response = api.NewInvokeResponse()
	req := fasthttp.AcquireRequest()

	endpoint := c.host
	if c.prefix != "" {
		endpoint += c.prefix
	}
	p := fmt.Sprintf("%s://%s/%s/functions/%s/invocations", c.scheme, endpoint, c.version, ir.FunctionBRN)
	//添加query参数
	if ir.Queries != nil && len(ir.Queries.Encode()) > 0 {
		p += "?" + ir.Queries.Encode()
	}
	req.SetRequestURI(p)

	if ir.Headers != nil {
		for k, v := range ir.Headers {
			req.Header.Set(k, v)
		}
	}
	req.Header.SetHost(c.host)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Set(api.HeaderXRequestID, ir.RequestID)
	req.Header.Set(api.HeaderXAccountID, ir.UserID)
	if ir.Authorization != "" {
		req.Header.Set(api.HeaderAuthorization, ir.Authorization)
	}
	req.Header.Set(api.HeaderInvokeType, ir.InvokeType)
	req.Header.Set(api.BceFaasTriggerKey, ir.TriggerType)
	req.Header.Set(api.HeaderLogType, string(ir.LogType))
	if ir.LogToBody {
		req.Header.Set(api.HeaderLogToBody, "true")
	}
	if ir.Body != nil {
		req.SetBodyString(*ir.Body)
	}

	if ir.Body != nil {
		req.SetBodyString(*ir.Body)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		logs.Errorf("unexpected minikun client error: %s", err)
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

func (c *ControllerClient) ListRuntimes() ([]*rtctrl.RuntimeInfo, error) {
	rts := make([]*rtctrl.RuntimeInfo, 0)
	req := fasthttp.AcquireRequest()
	p := fmt.Sprintf("%s://%s/%s/runtimes", c.scheme, c.host, c.version)
	req.SetRequestURI(p)

	req.Header.SetHost(c.host)
	req.Header.SetMethod("GET")
	req.Header.SetContentType("application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		logs.Errorf("unexpected minikun client error: %s", err)
		// TODO: handler unexpected error
		return nil, err
	}
	err = json.Unmarshal(resp.Body(), &rts)
	if err != nil {
		logs.Errorf("json unmarshal err :%v", err)
		return nil, err
	}
	return rts, nil
}
