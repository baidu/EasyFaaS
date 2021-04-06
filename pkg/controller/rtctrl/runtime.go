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
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	innerErr "github.com/baidu/openless/pkg/error"
	"github.com/baidu/openless/pkg/util/json"
	"github.com/baidu/openless/pkg/util/logs"
)

var (
	httpRetryDuration = time.Duration(100) * time.Millisecond
)

const (
	RuntimeHTTPSock = ".runtime-http.sock"
)

// initRuntime
func (info *RuntimeInfo) initRuntime(params *startRuntimeParams) error {
	preInit, _ := strconv.ParseInt(params.urlParams.Get("initstart"), 10, 64)
	postInit, _ := strconv.ParseInt(params.urlParams.Get("initdone"), 10, 64)

	info.invokeLock.Lock()
	defer info.invokeLock.Unlock()

	if info.State != RuntimeStateCold && info.State != RuntimeStateWarmUp {
		logs.Errorf("runtime %s  current states is %s", info.RuntimeID, info.State)
		return fmt.Errorf("duplicate runtime")
	}

	// TODO: CommitID could be missing, when controller restarted
	if len(info.CommitID) == 0 {
		info.SetCommitID(params.commitID)
	}
	if info.CommitID != params.commitID {
		return fmt.Errorf("commit id %s is not equal to %s", info.CommitID, params.commitID)
	}

	cm := params.urlParams.Get("concurrentmode")
	// When service's concurrent mode is true, the value of runtime concurrent mode makes sense
	if info.ConcurrentMode == true && cm == "false" {
		info.ConcurrentMode = false
	}
	logs.V(9).Infof("runtime[%s] concurrentMode is %t", info.RuntimeID, info.ConcurrentMode)

	info.SetInitTime(preInit, postInit)
	info.SetState(RuntimeStateWarm)
	info.SetUsed(true)
	if params.warmNotify != nil {
		close(params.warmNotify)
	}
	// when runtime restart
	// requestChan and runtimeStopChan should be reset
	if info.WithStreamMode {
		info.httpRequestChan = make(chan *InvokeHTTPRequest, 100)
		info.httpResponseChan = make(chan *InvokeHTTPResponse, 100)
		info.retryDeadline = time.Duration(info.WaitRuntimeAliveTimeout) * time.Second
	} else {
		info.requestChan = make(chan *InvokeRequest, 100)
	}
	info.runtimeStopChan = make(chan struct{})
	info.runtimeStoppingChan = make(chan struct{})

	info.sendRequestLoop(params)
	close(info.runtimeRunChan)
	return nil
}

// startRuntimeLoop
func (info *RuntimeInfo) startRuntimeLoop(params *startRuntimeParams) error {
	if err := info.initRuntime(params); err != nil {
		return err
	}

	info.recvResponseLoop()
	return nil
}

// sendRequestLoop
func (info *RuntimeInfo) sendRequestLoop(params *startRuntimeParams) {
	if !info.WithStreamMode {
		info.runtimeConn = params.conn
		info.runtimeWaitGroup.Add(1)
		go info.sendGenericRequestLoop()
	} else {
		if info.httpClient == nil {
			info.httpClient = info.getHTTPClient()
		}
		info.runtimeWaitGroup.Add(1)
		go info.sendHTTPRequestLoop()
	}
}

// sendGenericRequestLoop
func (info *RuntimeInfo) sendGenericRequestLoop() {
	encoder := json.NewEncoder(info.runtimeConn)
loop:
	for {
		select {
		case request := <-info.requestChan:
			err := encoder.Encode(request)
			requestInfo := info.loadRequest(request.RequestID)
			if err != nil {
				logs.Errorf("marshal invokeReq failed. %s", err.Error())
				if requestInfo != nil {
					requestInfo.InvokeResult(StatusFailed,
						fmt.Sprintf("RequestID: %s Process exited before completing request", request.RequestID))
					requestInfo.Notify()
				}
			}
			if requestInfo != nil {
				requestInfo.StepDone(StageSendRequest)
			}
		case <-info.runtimeStopChan:
			break loop
		}
	}

	info.runtimeWaitGroup.Done()
	logs.Infof("runtime %s stop sending request", info.RuntimeID)
}

func (info *RuntimeInfo) sendHTTPRequestLoop() {
	needStop := false
	defer func() {
		if needStop {
			info.stopRuntime()
		}
	}()
loop:
	for {
		select {
		case request := <-info.httpRequestChan:

			httpreq := request.Request
			reqLogger := request.Logger
			requestInfo := info.loadRequest(request.RequestID)
			if requestInfo != nil {
				requestInfo.StepDone(StageSendRequest)
			}
			cancel := request.CtxCancel
			retryDuration := httpRetryDuration
			timer := time.NewTimer(info.retryDeadline)
		RequestLoop:
			for {
				select {
				case <-timer.C:
					errMsg := fmt.Sprintf("%s Task timed out after %d seconds", requestInfo.RequestID, info.WaitRuntimeAliveTimeout)
					m := innerErr.AwsErrorMessage{
						ErrorMessage: errMsg,
					}
					requestInfo.InvokeResult(StatusTimeout, m.String())
					requestInfo.Notify()
					break RequestLoop
				case <-requestInfo.TimeoutChannel:
					// when request timeout, cancel the request
					cancel()
					break RequestLoop
				default:
					invokeStart := time.Now().UnixNano()
					httprsp, err := info.httpClient.Do(httpreq)
					invokeEnd := time.Now().UnixNano()
					if err != nil {
						reqLogger.Infof("request failed: %s retry after %s", err.Error(), retryDuration.String())
						<-time.After(retryDuration)
						if retryDuration < time.Second {
							retryDuration += time.Duration(100) * time.Millisecond
						}
						continue
					}
					requestInfo.InvokeDone()
					requestInfo.StepDone(StageInvokeDone)
					reqLogger.Infof("invoke time %d", time.Duration(invokeEnd-invokeStart)/time.Microsecond)
					resp := &InvokeHTTPResponse{
						RequestID: request.RequestID,
						Response:  httprsp,
					}
					info.httpResponseChan <- resp
					break RequestLoop
				}
			}
			if requestInfo != nil {
				requestInfo.StepDone(StageRecvResponse)
			}
			httpreq.Close = true
			timer.Stop()
		case <-info.runtimeStoppingChan:
			needStop = true
			break loop
		case <-info.runtimeStopChan:
			break loop
		}
	}
	info.runtimeWaitGroup.Done()
	logs.Infof("runtime %s stop sending http request", info.RuntimeID)
}

func (info *RuntimeInfo) recvResponseLoop() {
	if info.WithStreamMode {
		info.runtimeWaitGroup.Add(1)
		info.recvHTTPResponseLoop()
	} else {
		info.runtimeWaitGroup.Add(1)
		info.recvGenericResponseLoop()
	}
}

func (info *RuntimeInfo) recvGenericResponseLoop() {
	defer info.stopRuntime()

	urlParams := make(url.Values)
	decoder := json.NewDecoder(info.runtimeConn)
	for {
		var output InvokeResponse
		err := decoder.Decode(&output)
		if err != nil {
			if err == io.EOF {
				logs.Infof("runtime %s read EOF", info.RuntimeID)
			} else {
				logs.Errorf("decode %s response error %s", info.RuntimeID, err.Error())
			}
			break
		}
		logs.V(6).Info("request done.",
			zap.String("runtimeID", info.RuntimeID),
			zap.String("request_id", output.RequestID),
			zap.Bool("success", output.Success))
		requestInfo := info.loadRequest(output.RequestID)
		if requestInfo != nil {
			requestInfo.StepDone(StageRecvResponse)
		}
		if output.Success {
			urlParams.Set("success", "true")
			info.handleInvokeDone(output.RequestID, &urlParams, output.FuncResult)
		} else {
			urlParams.Set("success", "false")
			info.handleInvokeDone(output.RequestID, &urlParams, output.FuncError)
		}
	}

	info.runtimeWaitGroup.Done()
	logs.Infof("runtime %s stop receiving response", info.RuntimeID)
}

func (info *RuntimeInfo) recvHTTPResponseLoop() {
L:
	for {
		select {
		case response := <-info.httpResponseChan:
			httpResponse := response.Response
			requestInfo := info.loadRequest(response.RequestID)
			ConvertHTTPResponseToProxy(httpResponse, requestInfo)
			info.InvokeDone(requestInfo, true)
		case <-info.runtimeStopChan:
			break L
		}
	}
	info.runtimeWaitGroup.Done()
	logs.Infof("runtime %s stop receiving http response", info.RuntimeID)
}

func (info *RuntimeInfo) loadRequest(requestID string) *RequestInfo {
	if value, load := info.requestMap.Load(requestID); load {
		return value.(*RequestInfo)
	}
	logs.Error(fmt.Sprintf("request not found in %s runtimeDispatcher", info.RuntimeID),
		zap.String("request_id", requestID))
	return nil
}

func (info *RuntimeInfo) handleInvokeDone(requestID string, params *url.Values, data string) bool {
	if request := info.loadRequest(requestID); request != nil {
		status := StatusFailed
		if strings.Compare(params.Get("success"), "true") == 0 {
			status = StatusSuccess
		}
		request.InvokeResult(status, data)
		request.InvokeDone()
		request.StepDone(StageInvokeDone)
		info.InvokeDone(request, true)
		return true
	}

	return false
}

func (info *RuntimeInfo) InvokeDone(request *RequestInfo, signal bool) {
	if _, load := info.requestMap.Load(request.RequestID); !load {
		return
	}

	if signal {
		request.Notify()
	}

	if request.Status == StatusSuccess {
		info.AcceptReqCnt++
	} else if request.Status == StatusFailed {
		info.RejectReqCnt++
	}

	info.requestMap.Delete(request.RequestID)
}

// Wait
func (info *RuntimeInfo) Wait(timeout int) bool {
	if info.State == RuntimeStateWarm {
		return true
	}

	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	select {
	case <-info.runtimeRunChan:
	case <-timer.C:
	}
	timer.Stop()

	if info.State == RuntimeStateWarm {
		return true
	}

	return false
}

// stopRuntime
func (info *RuntimeInfo) stopRuntime() {
	info.invokeLock.Lock()
	defer info.invokeLock.Unlock()

	logs.V(5).Infof("close runtime %s", info.RuntimeID)

	info.PreLoadTimeMS = 0
	info.PostLoadTimeMS = 0
	info.PreInitTimeMS = 0
	info.PostInitTimeMS = 0
	info.AcceptReqCnt = 0
	info.RejectReqCnt = 0
	info.Concurrency = 0

	info.requestMap.Range(func(key, value interface{}) bool {
		id := key.(string)
		request := value.(*RequestInfo)
		request.InvokeResult(StatusFailed, fmt.Sprintf("RequestID: %s Process exited before completing request", id))
		request.Notify()
		return true
	})
	info.requestMap = sync.Map{}
	close(info.runtimeStopChan)
	if !info.WithStreamMode {
		info.runtimeConn.Close()
	} else {
		info.httpClient = nil
	}
	info.runtimeWaitGroup.Wait()
	info.updateStreamMode(false)

	info.SetState(RuntimeStateStopped)

	info.runtimeRunChan = make(chan struct{})
}

// InvokeFunc
func (info *RuntimeInfo) InvokeFunc(request *RequestInfo, invokeReq *InvokeRequest) error {
	info.requestMap.Store(request.RequestID, request)

	select {
	case info.requestChan <- invokeReq:
	default:
		return errors.New("request queue is full")
	}

	return nil
}

// InvokeFunc
func (info *RuntimeInfo) InvokeHTTPFunc(request *RequestInfo, invokeReq *InvokeHTTPRequest) error {
	info.requestMap.Store(request.RequestID, request)

	select {
	case info.httpRequestChan <- invokeReq:
	default:
		return errors.New("request queue is full")
	}

	return nil
}

// TODO: Add a timer to recreate the client ?
func (info *RuntimeInfo) getHTTPClient() *http.Client {
	sock := fmt.Sprintf("/var/run/faas/%s", info.RuntimeID)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", fmt.Sprintf("%s/%s", sock, RuntimeHTTPSock))
			},
		},
		// forbid redirect
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return client
}
