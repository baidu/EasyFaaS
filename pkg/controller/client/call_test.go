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

// Package client
package client

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/controller"

	"github.com/baidu/easyfaas/cmd/controller/options"
)

func TestControllerCallClient_Invoke(t *testing.T) {
	opt := options.NewOptions()
	mc := &MockController{}
	client := NewControllerCallClient(mc, nil, opt)
	bodyStr := "{\"k1\":\"v1\"}"
	ir := &api.InvokeRequest{
		UserID:      "df391b08c64c426a81645468c75163a5",
		FunctionBRN: "brn:cloud:faas:bj:cd64f99c69d7c404b61de0a4f1865834:function:concurrentHello:1",
		Body:        &bodyStr,
		RequestID:   "xxx",
	}
	client.Invoke(ir)

	read := bytes.NewBuffer(nil)
	write := bytes.NewBuffer(nil)
	bodyStream := bufio.NewReadWriter(bufio.NewReader(read), bufio.NewWriter(write))
	ir2 := &api.InvokeRequest{
		UserID:         "df391b08c64c426a81645468c75163a5",
		FunctionBRN:    "brn:cloud:faas:bj:cd64f99c69d7c404b61de0a4f1865834:function:concurrentHello:1",
		BodyStream:     bodyStream,
		RequestID:      "xxx",
		WithBodyStream: true,
	}
	client.Invoke(ir2)
	return
}

type MockController struct{}

func (mm *MockController) NewClients(string) *controller.Clients { return nil }

func (mm *MockController) Do(ctx *controller.InvokeContext) { return }
