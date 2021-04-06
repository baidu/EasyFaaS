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

package rtctrl

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/baidu/openless/pkg/api"

	"go.uber.org/zap"

	innerErr "github.com/baidu/openless/pkg/error"
	"github.com/baidu/openless/pkg/util/logs"
)

type Control interface {
	InvokeFunction(input *InvocationInput) *InvocationOutput
}

type RuntimeClient struct {
	config         *RuntimeConfigOptions
	userlogType    UserLogType
	dispatchServer *DispatchServerV2
}

func NewRuntimeClient(c *RuntimeConfigOptions, s *DispatcherV2Options, rtMap RuntimeDispatcher) (rc *RuntimeClient, err error) {
	dispatchServer := NewDispatchServerV2(s, rtMap)
	dispatchServer.ListenAndServe()
	lt := UserLogType(s.UserLogType)
	if !lt.Valid() {
		err = fmt.Errorf("log type [%s] is invalid, should be plain or json", lt)
		return nil, err
	}
	return &RuntimeClient{
		dispatchServer: dispatchServer,
		config:         c,
		userlogType:    lt,
	}, nil
}

func (s *RuntimeClient) createRequest(input *InvocationInput) *RequestInfo {
	logs.V(5).Info("recv request.", zap.String("runtimeID", input.Runtime.RuntimeID))

	if input.Configuration.FunctionArn == nil {
		defaultFunctionArn := ""
		input.Configuration.FunctionArn = &defaultFunctionArn
	}
	if input.Configuration.Timeout == nil || *(input.Configuration.Timeout) == 0 {
		var dafaultTimeout int64 = 3
		input.Configuration.Timeout = &dafaultTimeout
	}

	reqinfo := NewRequestInfo(input.RequestID, input.Runtime)
	reqinfo.TriggerType = input.TriggerType
	reqinfo.Input = input
	reqinfo.Output = &InvocationOutput{
		Output:    &InvocationResponse{Response: input.Response},
		Statistic: &InvocationStatistic{},
	}
	if input.EnableMetrics {
		reqinfo.Output.Statistic.Metric = NewRtCtrlInvokeMetric(reqinfo.RequestID)
	}
	return reqinfo
}

func (s *RuntimeClient) makeInvokeRequest(input *InvocationInput) *InvokeRequest {
	invokeReq := &InvokeRequest{
		RequestID: input.RequestID,
		Version:   *input.Configuration.Version,
	}

	if input.InvokeType != api.InvokeTypeEvent && input.Request.BodyStream != nil {
		bs, err := ioutil.ReadAll(input.Request.BodyStream)
		if err != nil {
			input.Logger.Errorf("read event body failed: %s", err)
		} else {
			invokeReq.EventObject = string(bs)
		}
	} else {
		invokeReq.EventObject = string(input.Request.Body)
	}
	if invokeReq.EventObject == "" {
		invokeReq.EventObject = "{}"
	}
	return invokeReq
}

func (s *RuntimeClient) makeInvokeHTTPRequest(reqInfo *RequestInfo, input *InvocationInput) *InvokeHTTPRequest {
	req, cancel := ConvertProxyRequestToHTTP(reqInfo)
	invokeReq := &InvokeHTTPRequest{
		RequestID:       input.RequestID,
		Request:         req,
		FunctionTimeout: *input.Configuration.Timeout,
		Logger:          input.Logger,
		CtxCancel:       cancel,
	}
	return invokeReq
}

func (s *RuntimeClient) waitRuntime(reqinfo *RequestInfo) (err error) {
	runtime := reqinfo.Runtime
	input := reqinfo.Input
	ok := runtime.Wait(s.config.WaitRuntimeAliveTimeout)
	if !ok {
		err = fmt.Errorf("runtime %s not ready", runtime.RuntimeID)
		return
	}

	if input.Configuration.CommitID != nil &&
		*(input.Configuration.CommitID) != runtime.CommitID {
		err = fmt.Errorf("commitid [%s] [%s] mismatch, ", *input.Configuration.CommitID, runtime.CommitID)
		return
	}

	if input.Configuration.MemorySize != nil {
		reqinfo.MemorySpecSize = *(input.Configuration.MemorySize)
	} else {
		reqinfo.MemorySpecSize = api.MinMemorySize
	}
	runtime.UserID = input.User.ID
	return
}

func (s *RuntimeClient) invokeCleanup(reqinfo *RequestInfo, runtime *RuntimeInfo) {
	runtime.InvokeDone(reqinfo, false)
}

func (s *RuntimeClient) InvokeFunction(input *InvocationInput) (output *InvocationOutput) {
	var errorMessage string
	reqInfo := s.createRequest(input)

	defer func() {
		if errorMessage != "" {
			input.Logger.Errorf("invoke function failed, error is %s", errorMessage)
			m := innerErr.AwsErrorMessage{
				ErrorMessage: errorMessage,
			}
			reqInfo.Output.Output.FuncResult = m.String()
		}
		output = reqInfo.Output
	}()

	err := s.waitRuntime(reqInfo)
	if err != nil {
		errorMessage = fmt.Sprintf("%s wait runtime error: %s", reqInfo.RequestID, err.Error())
		return
	}
	reqInfo.StepDone(StageWaitRuntime)

	defer func() {
		s.invokeCleanup(reqInfo, input.Runtime)
		reqInfo.StepDone(StageCleanup)
	}()
	var logType string
	if !api.IsNoneLogType(reqInfo.Input.Configuration.LogType) ||
		(reqInfo.Input.LogConfig != nil && !api.IsNoneLogType(reqInfo.Input.LogConfig.LogType)) {
		logType = string(s.userlogType)
		reqInfo.enableUserLog = true
	}

	s.startRecvLog(reqInfo, logType)
	reqInfo.StepDone(StageStartRecvLog)
	reqInfo.Status = StatusRunning
	functionTimeout := int(*(input.Configuration.Timeout))
	err = s.InvokeFunc(reqInfo, input)

	timeout := false
	if err != nil {
		errorMessage = fmt.Sprintf("%s invoke function error: %s", reqInfo.RequestID, err.Error())
		reqInfo.InvokeDone()
		reqInfo.StepDone(StageInvokeDone)
		reqInfo.InvokeReportDone()
		reqInfo.StepDone(StageInvokeReportDone)
		s.dispatchServer.StopRecvLog(reqInfo.Runtime.RuntimeID, reqInfo.RequestID, reqInfo.store)
		reqInfo.StepDone(StageStopRecvLog)
		return
	}

	timer := time.NewTimer(time.Duration(functionTimeout) * time.Second)
	select {
	case <-reqInfo.SyncChannel:

	case <-timer.C:
		reqInfo.InvokeResult(StatusTimeout, "Invoke timeout.")
		if input.WithStreamMode {
			reqInfo.TimeoutChannel <- struct{}{}
		}
		timeout = true
	}
	timer.Stop()

	reqInfo.InvokeReportDone()
	reqInfo.StepDone(StageInvokeReportDone)
	s.dispatchServer.StopRecvLog(reqInfo.Runtime.RuntimeID, reqInfo.RequestID, reqInfo.store)
	reqInfo.StepDone(StageStopRecvLog)
	reqInfo.Output.Statistic.Statistic = statisticsInfo(reqInfo)
	if timeout {
		errorMessage = fmt.Sprintf("%s Task timed out after %d seconds", reqInfo.RequestID, functionTimeout)
		return
	}
	return
}

func (s *RuntimeClient) InvokeFunc(reqInfo *RequestInfo, input *InvocationInput) (err error) {
	if input.WithStreamMode {
		invokeRequest := s.makeInvokeHTTPRequest(reqInfo, input)
		reqInfo.InvokeStart()
		err = input.Runtime.InvokeHTTPFunc(reqInfo, invokeRequest)
		reqInfo.StepDone(StageInvokeFunc)
	} else {
		invokeReq := s.makeInvokeRequest(input)
		reqInfo.InvokeStart()
		err = input.Runtime.InvokeFunc(reqInfo, invokeReq)
		reqInfo.StepDone(StageInvokeFunc)
		reqInfo.CleanInput()
		invokeReq = nil
	}
	return
}

func (s *RuntimeClient) startRecvLog(reqInfo *RequestInfo, logType string) {
	userID := reqInfo.Runtime.UserID
	if userID == "" {
		logs.Warn("user id is empty, set to unknown")
		userID = "unknown"
	}
	params := LogStatStoreParameter{
		RequestID:       reqInfo.RequestID,
		TriggerType:     reqInfo.TriggerType,
		RuntimeID:       reqInfo.Runtime.RuntimeID,
		UserID:          userID,
		FunctionName:    *reqInfo.Input.Configuration.FunctionName,
		FunctionBrn:     *reqInfo.Input.Configuration.FunctionArn,
		FunctionVersion: *reqInfo.Input.Configuration.Version,
		FilePath:        s.dispatchServer.config.UserLogFileDir,
		LogType:         logType,
	}
	lss := newLogStatStore(&params)
	s.dispatchServer.StartRecvLog(reqInfo.Runtime.RuntimeID, reqInfo.RequestID, lss)
	reqInfo.SetLogStore(lss)
	return
}
