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

// Package rtctrl
package rtctrl

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/lambda"

	kunhttptest "github.com/baidu/easyfaas/pkg/util/httptest"
	"github.com/baidu/easyfaas/pkg/util/id"
	"github.com/baidu/easyfaas/pkg/util/logs"

	"github.com/baidu/easyfaas/pkg/api"
)

func TestInvocation(t *testing.T) {
	rtID := "runtime"
	rtMap := NewRuntimeManager(getFuncletNode(), &RuntimeManagerParameters{MaxRuntimeIdle: 10, MaxRunnerDefunct: 30})
	rtParams := &NewRuntimeParameters{
		RuntimeID:               rtID,
		ConcurrentMode:          true,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rt := rtMap.NewRuntime(rtParams)
	rt.SetState(RuntimeStateWarm)

	rc := NewRuntimeConfigOptions()
	ds := &DispatcherV2Options{
		RunnerServerAddress:  getTmpSock(),
		RuntimeServerAddress: getTmpSock(),
		UserLogFileDir:       defaultUserLogFilePath,
		UserLogType:          string(UserLogTypePlain),
	}
	cli, _ := NewRuntimeClient(rc, ds, rtMap)

	<-time.NewTicker(time.Second).C
	funN := "test"
	cmID := "xxx"
	timeout := int64(3)
	mem := int64(256)
	ver := "1"
	reqid := id.GetRequestID()
	rt.SetCommitID(cmID)
	go initRuntime(cli.dispatchServer, rt.RuntimeID, cmID, reqid)

	input := &InvocationInput{
		Runtime:   rt,
		RequestID: reqid,
		Configuration: &api.FunctionConfiguration{
			CommitID: &cmID,
			FunctionConfiguration: lambda.FunctionConfiguration{
				FunctionName: &funN,
				Timeout:      &timeout,
				MemorySize:   &mem,
				Version:      &ver,
			},
		},
		User: &api.User{
			ID: "xxx",
		},
		WithStreamMode: false,
		Request: &api.InvokeProxyRequest{
			Headers: make(map[string]string, 0),
			Body:    []byte("test"),
		},
		Response:      api.NewInvokeProxyResponse(),
		EnableMetrics: false,
		Logger:        logs.NewLogger().WithField("request_id", reqid),
	}

	output := cli.InvokeFunction(input)

	t.Logf("%+v", output)
}

func TestInvocationWithStream(t *testing.T) {
	rtID := "runtime"
	rtMap := NewRuntimeManager(getFuncletNode(), &RuntimeManagerParameters{MaxRuntimeIdle: 10, MaxRunnerDefunct: 30})
	rtParams := &NewRuntimeParameters{
		RuntimeID:               rtID,
		ConcurrentMode:          true,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rt := rtMap.NewRuntime(rtParams)
	rt.SetState(RuntimeStateWarm)

	rc := NewRuntimeConfigOptions()
	ds := &DispatcherV2Options{
		RunnerServerAddress:  getTmpSock(),
		RuntimeServerAddress: getTmpSock(),
		UserLogFileDir:       defaultUserLogFilePath,
		UserLogType:          string(UserLogTypePlain),
	}
	cli, _ := NewRuntimeClient(rc, ds, rtMap)

	<-time.NewTicker(time.Second).C
	cmID := "xxx"
	ver := "1"
	funN := "test"
	reqid := id.GetRequestID()
	rt.SetCommitID(cmID)
	rt.updateStreamMode(true)
	go initRuntime(cli.dispatchServer, rt.RuntimeID, cmID, reqid)

	input := &InvocationInput{
		Runtime:   rt,
		RequestID: reqid,
		Configuration: &api.FunctionConfiguration{
			CommitID: &cmID,
			FunctionConfiguration: lambda.FunctionConfiguration{
				FunctionName: &funN,
				Version:      &ver,
			},
		},
		User: &api.User{
			ID: "xxx",
		},
		WithStreamMode: true,
		Request: &api.InvokeProxyRequest{
			Headers: make(map[string]string, 0),
			Body:    []byte("test"),
		},
		Response:      api.NewInvokeProxyResponse(),
		EnableMetrics: true,
		Logger:        logs.NewLogger().WithField("request_id", reqid),
	}
	output := cli.InvokeFunction(input)
	t.Logf("%+v", output)
}

func initRuntime(s *DispatchServerV2, runtimeID string, commitID string, requestID string) {
	body := `{"key1":1,"key2":2}`
	req := httptest.NewRequest("POST", "/status", strings.NewReader(body))
	req.Header.Set("x-cfc-runtimeid", runtimeID)
	req.Header.Set("x-cfc-commitid", commitID)
	req.Header.Set("x-cfc-hostip", "127.0.0.1")
	q := req.URL.Query()
	q.Add("initstart", "10")
	q.Add("initdone", "20")
	q.Add("concurrentmode", "false")
	req.URL.RawQuery = q.Encode()
	bodyString := fmt.Sprintf("{\"requestid\":\"%s\", \"success\": \"true\"}\n", requestID)
	bodyBytes := []byte(bodyString)
	resp := kunhttptest.NewResponseHijacker(bodyBytes)

	s.invokeHandler(resp, req)
}
