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
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	utilID "github.com/baidu/easyfaas/pkg/util/id"

	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/util/logs"

	kunhttptest "github.com/baidu/easyfaas/pkg/util/httptest"
)

func TestSendHTTPRequestLoop(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
		return
	}))
	defer ts.Close()

	id := "runtime-id"
	cmid := "commit-xxx"
	reqid := utilID.GetRequestID()
	rtMap := NewRuntimeManager(getFuncletNode(), &RuntimeManagerParameters{MaxRuntimeIdle: 10, MaxRunnerDefunct: 30})
	rtParams := &NewRuntimeParameters{
		RuntimeID:               id,
		ConcurrentMode:          false,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rt := rtMap.NewRuntime(rtParams)
	req, resp := buildRequestResponse(id, cmid, reqid)
	conn, _, ok := setupHijackConn(resp, req)
	if !ok {
		t.Error("setup hijack connect failed")
		return
	}
	rt.SetState(RuntimeStateCold)
	rt.updateStreamMode(true)
	params := &startRuntimeParams{
		commitID:   cmid,
		hostIP:     "127.0.0.1",
		conn:       conn,
		warmNotify: make(chan struct{}),
		urlParams:  req.URL.Query(),
	}
	if err := rt.initRuntime(params); err != nil {
		t.Errorf("init runtime failed: %s", err)
		return
	}
	rt.httpClient = ts.Client()

	now := time.Now().UnixNano()
	ver := "1"
	timeout := int64(3)
	reqinfo := RequestInfo{
		RequestID:         reqid,
		Runtime:           rt,
		InvokeStartTimeNS: now,
		InvokeStartTimeMS: now / int64(time.Millisecond),
		SyncChannel:       make(chan struct{}, 1),
		TimeoutChannel:    make(chan struct{}, 1),
		Input: &InvocationInput{
			Runtime:   rt,
			RequestID: reqid,
			Configuration: &api.FunctionConfiguration{
				CommitID: &cmid,
				FunctionConfiguration: lambda.FunctionConfiguration{
					Version: &ver,
					Timeout: &timeout,
				},
			},
			User: &api.User{
				ID: "xxx",
			},
			WithStreamMode: true,
			Request: &api.InvokeProxyRequest{
				Headers: make(map[string]string, 0),
				Body:    []byte("{}"),
			},
			Response:      api.NewInvokeProxyResponse(),
			EnableMetrics: false,
			Logger:        logs.NewLogger().WithField("request_id", reqid),
		},
		Output: &InvocationOutput{Output: &InvocationResponse{}, Statistic: &InvocationStatistic{}},
	}
	reqinfo.Output.Output.Response = reqinfo.Input.Response
	req, cancel := ConvertProxyRequestToHTTP(&reqinfo)
	req.URL, _ = url.Parse(ts.URL)
	invokeReq := InvokeHTTPRequest{
		RequestID:       reqinfo.RequestID,
		Request:         req,
		FunctionTimeout: *reqinfo.Input.Configuration.Timeout,
		Logger:          reqinfo.Input.Logger,
		CtxCancel:       cancel,
	}

	err := rt.InvokeHTTPFunc(&reqinfo, &invokeReq)
	if err != nil {
		t.Errorf("send request failed: %s", err)
		return
	}
	rt.runtimeWaitGroup.Add(1)
	go rt.recvHTTPResponseLoop()
	ticker := time.NewTicker(3 * time.Second)
L:
	for {
		select {
		case <-reqinfo.SyncChannel:
			break L
		case <-ticker.C:
			t.Errorf("expect invoke success, but got timeout")
			break L
		}
	}
	close(rt.runtimeStoppingChan)
	return
}

func TestSendGenericRequestLoop(t *testing.T) {
	id := "runtime-id"
	cmid := "commit-xxx"
	reqid := utilID.GetRequestID()
	rtMap := NewRuntimeManager(getFuncletNode(), &RuntimeManagerParameters{MaxRuntimeIdle: 10, MaxRunnerDefunct: 30})
	rtParams := &NewRuntimeParameters{
		RuntimeID:               id,
		ConcurrentMode:          false,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rt := rtMap.NewRuntime(rtParams)
	req, resp := buildRequestResponse(id, cmid, reqid)
	conn, _, ok := setupHijackConn(resp, req)
	if !ok {
		t.Error("setup hijack connect failed")
		return
	}
	rt.SetState(RuntimeStateCold)
	params := &startRuntimeParams{
		commitID:   cmid,
		hostIP:     "127.0.0.1",
		conn:       conn,
		warmNotify: make(chan struct{}),
		urlParams:  req.URL.Query(),
	}
	if err := rt.initRuntime(params); err != nil {
		t.Errorf("init runtime failed: %s", err)
		return
	}

	now := time.Now().UnixNano()
	ver := "1"
	reqinfo := RequestInfo{
		RequestID:         reqid,
		Runtime:           rt,
		InvokeStartTimeNS: now,
		InvokeStartTimeMS: now / int64(time.Millisecond),
		SyncChannel:       make(chan struct{}, 1),
		TimeoutChannel:    make(chan struct{}, 1),
		Input: &InvocationInput{
			Runtime:   rt,
			RequestID: reqid,
			Configuration: &api.FunctionConfiguration{
				CommitID:              &cmid,
				FunctionConfiguration: lambda.FunctionConfiguration{Version: &ver},
			},
			User: &api.User{
				ID: "xxx",
			},
			WithStreamMode: false,
			Request: &api.InvokeProxyRequest{
				Headers: make(map[string]string, 0),
				Body:    []byte("{}"),
			},
			Response:      api.NewInvokeProxyResponse(),
			EnableMetrics: true,
			Logger:        logs.NewLogger().WithField("request_id", reqid),
		},
		Output: &InvocationOutput{Output: &InvocationResponse{}, Statistic: &InvocationStatistic{
			Metric: NewRtCtrlInvokeMetric(reqid),
		}},
	}
	invokeReq := &InvokeRequest{
		RequestID:   reqinfo.RequestID,
		Version:     *reqinfo.Input.Configuration.Version,
		EventObject: string(reqinfo.Input.Request.Body),
	}

	err := rt.InvokeFunc(&reqinfo, invokeReq)
	if err != nil {
		t.Errorf("send request failed: %s", err)
		return
	}
	rt.runtimeWaitGroup.Add(1)
	rt.recvGenericResponseLoop()
	ticker := time.NewTicker(3 * time.Second)
L:
	for {
		select {
		case <-reqinfo.SyncChannel:
			break L
		case <-ticker.C:
			t.Errorf("expect invoke success, but got timeout")
			break L
		}
	}
	close(rt.runtimeStoppingChan)
	return
}

func buildRequestResponse(runtimeID string, commitID string, requestID string) (req *http.Request, resp *kunhttptest.ResponseHijacker) {
	body := `{"key1":1,"key2":2}`
	req = httptest.NewRequest("POST", "/status", strings.NewReader(body))
	req.Header.Set("x-cfc-runtimeid", runtimeID)
	req.Header.Set("x-cfc-commitid", commitID)
	req.Header.Set("x-cfc-hostip", "127.0.0.1")
	q := req.URL.Query()
	q.Add("initstart", "10")
	q.Add("initdone", "20")
	q.Add("concurrentmode", "false")
	req.URL.RawQuery = q.Encode()
	bodyString := fmt.Sprintf("{\"requestid\":\"%s\", \"success\":true}", requestID)
	bodyBytes := []byte(bodyString)
	resp = kunhttptest.NewResponseHijacker(bodyBytes)
	return req, resp
}
