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
	"strings"
	"testing"

	"github.com/baidu/easyfaas/pkg/util/bytefmt"

	"github.com/stretchr/testify/assert"

	"github.com/baidu/easyfaas/pkg/api"

	"github.com/google/uuid"

	kunhttptest "github.com/baidu/easyfaas/pkg/util/httptest"
)

func getTmpSock() string {
	s := uuid.New().String()
	return fmt.Sprintf("unix:///tmp/%s.sock", s)
}

func TestListenAndServe(t *testing.T) {
	rtMap := initRuntimeList(10)
	runnerServerAddress := getTmpSock()
	runtimeServerAddress := getTmpSock()
	opt := &DispatcherV2Options{
		RunnerServerAddress:  runnerServerAddress,
		RuntimeServerAddress: runtimeServerAddress,
	}
	s := NewDispatchServerV2(opt, rtMap)
	s.ListenAndServe()
}

func TestRunnerHandler(t *testing.T) {
	rtMap := NewRuntimeManager(getFuncletNode(), &RuntimeManagerParameters{MaxRuntimeIdle: 10, MaxRunnerDefunct: 30})
	rtID := "runtime"
	rtParams := &NewRuntimeParameters{
		RuntimeID:               rtID,
		ConcurrentMode:          true,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rtMap.NewRuntime(rtParams)
	opt := &DispatcherV2Options{}
	s := NewDispatchServerV2(opt, rtMap)
	s.getRunnerRouteHandler()
	body := `{"key1":1,"key2":2}`
	req := httptest.NewRequest("POST", "/status", strings.NewReader(body))
	req.Header.Set("x-cfc-runtimeid", rtID)
	bodyBytes := make([]byte, 0)
	resp := kunhttptest.NewResponseHijacker(bodyBytes)
	s.runnerHandler(resp, req)
	rtinfo, _ := rtMap.GetRuntime(rtID)
	if rtinfo.State != RuntimeStateClosed {
		t.Errorf("runtime state should be closed, but got %s", rtinfo.State)
	}
	t.Logf("status code %d", resp.Code())
}

func TestStdlogHandler(t *testing.T) {
	rtMap := initRuntimeList(10)
	opt := &DispatcherV2Options{}
	defaultLogBufferLength = 70
	s := NewDispatchServerV2(opt, rtMap)
	s.getRuntimeRouteHandler()

	req, resp := buildStdlogRequestResponse()
	params1 := &LogStatStoreParameter{
		RequestID:       "reqid-1",
		RuntimeID:       "runtime-1",
		UserID:          "test-1",
		FunctionName:    "func-1",
		FunctionVersion: "1",
		FilePath:        "/tmp",
		LogType:         "bos",
	}
	store := newLogStatStore(params1)
	t.Logf("write log file %s\n", store.LogFile())
	s.StartRecvLog("runtime-1", "reqid-1", store)
	params2 := &LogStatStoreParameter{
		RequestID:       "reqid-2",
		RuntimeID:       "runtime-1",
		UserID:          "test-1",
		FunctionName:    "func-1",
		FunctionVersion: "1",
		FilePath:        "/tmp",
		LogType:         "bos",
	}
	store2 := newLogStatStore(params2)
	t.Logf("write log file %s\n", store.LogFile())
	s.StartRecvLog("runtime-1", "reqid-2", store2)
	stdoutFunc := s.stdlogHandler(StdoutLog)
	stdoutFunc(resp, req)
	if resp.Code() != http.StatusOK {
		t.Errorf("status code %d", resp.Code())
	}
}

func buildStdlogRequestResponse() (req *http.Request, resp *kunhttptest.ResponseHijacker) {
	body := `{"key1":1,"key2":2}`
	req = httptest.NewRequest("POST", "/stdout", strings.NewReader(body))
	req.Header.Set("x-cfc-runtimeid", "runtime-1")
	bodyBytes := []byte("2020-05-13T03:10:55.690Z\treqid-1\thello\n2020-05-13T03:10:55.690Z\treqid-1\thello tmp\n2020-05-13T03:10:55.690Z\treqid-2\thello\ntest\n2020-05-13T03:10:55.690Z\treqid-2\thello tmp\n\0002020-05-13T03:10:55.690Z\treqid-1\thello tmp req-1\n\000")
	resp = kunhttptest.NewResponseHijacker(bodyBytes)
	return
}

func TestInvokeHandler(t *testing.T) {
	rtMap := initRuntimeList(10)
	opt := &DispatcherV2Options{}
	s := NewDispatchServerV2(opt, rtMap)
	s.getRuntimeRouteHandler()

	body := `{"key1":1,"key2":2}`
	req := httptest.NewRequest("POST", "/status", strings.NewReader(body))
	req.Header.Set("x-cfc-runtimeid", "runtime-1")
	req.Header.Set("x-cfc-commitid", "123")
	req.Header.Set("x-cfc-hostip", "127.0.0.1")
	q := req.URL.Query()
	q.Add("initstart", "10")
	q.Add("initdone", "20")
	q.Add("concurrentmode", "false")
	req.URL.RawQuery = q.Encode()
	bodyBytes := []byte(`{"requestid":"xx", "success": "true"}`)
	resp := kunhttptest.NewResponseHijacker(bodyBytes)
	ors := s.runtimeDispatcher.ResourceStatistics()
	s.invokeHandler(resp, req)
	t.Logf("status code %d", resp.Code())
	or := s.runtimeDispatcher.ResourceStatistics()
	assert.Equal(t, or.Used.Memory-ors.Used.Memory, minMemory*bytefmt.Megabyte)
}

func TestStatisticHandler(t *testing.T) {
	rtMap := NewRuntimeManager(getFuncletNode(), &RuntimeManagerParameters{MaxRuntimeIdle: 10, MaxRunnerDefunct: 30})
	rtID := "runtime"
	rtParams := &NewRuntimeParameters{
		RuntimeID:               rtID,
		ConcurrentMode:          true,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rtMap.NewRuntime(rtParams)
	opt := &DispatcherV2Options{}
	s := NewDispatchServerV2(opt, rtMap)
	s.getRuntimeRouteHandler()

	body := `{"key1":1,"key2":2}`
	req := httptest.NewRequest("POST", "/statistic", strings.NewReader(body))
	req.Header.Set("x-cfc-runtimeid", rtID)
	bodyBytes := []byte(`{"podname": "runtime", "memory":10}`)
	resp := kunhttptest.NewResponseHijacker(bodyBytes)
	reqID := "xxxxx-xxx"
	params := &LogStatStoreParameter{
		RequestID:       reqID,
		RuntimeID:       rtID,
		UserID:          "user",
		FunctionName:    "func-1",
		FunctionVersion: "1",
		FilePath:        "/tmp",
		LogType:         "bos",
	}
	store := newLogStatStore(params)
	s.StartRecvLog(rtID, reqID, store)
	s.statisticHandler(resp, req)
	st := s.storeMap.get(rtID, reqID)
	if st.MemUsed() != int64(10) {
		t.Errorf("memused should be 10, but got %d", st.MemUsed())
	}
	t.Logf("status code %d", resp.Code())
}
