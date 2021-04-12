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
	"net/http"
	"time"

	"github.com/baidu/easyfaas/pkg/util/bytefmt"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

const (
	baseDuration time.Duration = 100 * time.Millisecond
	maxDuration  time.Duration = 1<<63 - 1
)

type RequestStatus int

const (
	StatusNormal RequestStatus = iota
	StatusRunning
	StatusSuccess
	StatusFailed
	StatusTimeout
)

func ceilBillDuration(d time.Duration) time.Duration {
	r := d % baseDuration
	if r == 0 {
		return d
	}
	if d1 := d + baseDuration - r; d1 > d {
		return d1
	}
	return maxDuration
}

type RequestInfo struct {
	RequestID         string
	Runtime           *RuntimeInfo
	InvokeStartTimeNS int64
	InvokeStartTimeMS int64
	InitStartTimeMS   int64
	InitDoneTimeMS    int64
	InvokeDoneTimeMS  int64
	InvokeDurationMS  time.Duration
	BilledDurationMS  time.Duration
	MaxMemUsedBytes   int64
	MemorySpecSize    int64
	TriggerType       string
	enableUserLog     bool

	Status RequestStatus
	Input  *InvocationInput
	Output *InvocationOutput

	store LogStatStore

	SyncChannel    chan struct{}
	TimeoutChannel chan struct{} // timeout notification
}

func NewRequestInfo(requestID string, runtime *RuntimeInfo) *RequestInfo {
	now := time.Now().UnixNano()
	return &RequestInfo{
		RequestID:         requestID,
		Runtime:           runtime,
		InvokeStartTimeNS: now,
		InvokeStartTimeMS: now / int64(time.Millisecond),
		SyncChannel:       make(chan struct{}, 1),
		TimeoutChannel:    make(chan struct{}, 1),
	}
}

func (info *RequestInfo) SetInitTime(preInit, postInit int64) {
	info.InitStartTimeMS = preInit
	info.InitDoneTimeMS = postInit
}

func (info *RequestInfo) SetLogStore(store LogStatStore) {
	info.store = store
}

func (info *RequestInfo) InvokeStart() {
	info.store.WriteFunctionLog(fmt.Sprintf("START RequestID: %s Version: %s\n",
		info.RequestID, *info.Input.Configuration.Version))
}

func (info *RequestInfo) CleanInput() {
	input := info.Input
	if input != nil {
		input.Request.Body = make([]byte, 0)
	}
}

func (info *RequestInfo) CleanOutput() {
	info.Output.Output.FuncResult = ""
	info.Output.Output.FuncError = ""
}

func (info *RequestInfo) InvokeResult(status RequestStatus, result string) {
	if info.Status != StatusRunning {
		return
	}
	info.Status = status
	if status == StatusSuccess {
		info.Output.Output.FuncResult = result
	} else {
		info.Output.Output.FuncError = "Unhandled"
		info.Output.Output.ErrorInfo = result
	}
}

func (info *RequestInfo) InvokeDone() {
	info.InvokeDoneTimeMS = time.Now().UnixNano() / int64(time.Millisecond)
	info.InvokeDurationMS = (time.Duration)(info.InvokeDoneTimeMS-info.InvokeStartTimeMS) * time.Millisecond
	if info.store != nil {
		info.MaxMemUsedBytes = info.store.MemUsed()
	}
	if !info.Runtime.ConcurrentMode {
		billed := ceilBillDuration(info.InvokeDurationMS)
		info.BilledDurationMS = billed
	}
}

func (info *RequestInfo) InvokeReportDone() {
	if !info.Runtime.ConcurrentMode && (info.enableUserLog || info.Input.IsLogTail) {
		info.waitLogDone()
	} else {
		info.store.LogDone(true)
	}
	info.store.WriteFunctionLog(fmt.Sprintf("END RequestID: %s\tMemory Spec Size: %dMB\n", info.RequestID, info.MemorySpecSize))
	if info.Runtime != nil && info.Runtime.ConcurrentMode {
		info.InvokeDurationMS = 0
		info.BilledDurationMS = 0
		info.MaxMemUsedBytes = 0
	}
	params := &reportParameters{
		InvocationTime: int64(info.InvokeDurationMS),
		MemUsage:       info.MaxMemUsedBytes,
		Mode:           info.getInvokeMode(),
		Status:         info.getResponseStatus(),
	}
	info.store.WriteFunctionReportLog(fmt.Sprintf("REPORT RequestID: %s\tDuration: %s\tBilled Duration: %s\tMax Memory Used: %s",
		info.RequestID, info.InvokeDurationMS.String(), info.BilledDurationMS.String(), bytefmt.ByteSize(uint64(info.MaxMemUsedBytes))),
		params)
	logData, err := info.store.Close()
	if err != nil {
		logs.Errorf("close log store err: %s", err)
	}
	if info.Input.IsLogTail {
		info.Output.Output.LogMessage = append(info.Output.Output.LogMessage, logData)
	}
}

func (info *RequestInfo) getInvokeMode() string {
	if info.Runtime != nil && info.Runtime.ConcurrentMode {
		return "concurrent"
	}
	return "common"
}

func (info *RequestInfo) getResponseStatus() int {
	if info.Status == StatusSuccess {
		return http.StatusOK
	}
	return http.StatusInternalServerError
}
func (info *RequestInfo) waitLogDone() {
	store := info.store
	if store.LogDone(false) {
		return
	}

	go func() {
		// max waiting time = 10ms
		timer := time.NewTimer(10 * time.Millisecond)
		<-timer.C
		store.LogDone(true)
		timer.Stop()
	}()
	store.Wait()
}
func (info *RequestInfo) Notify() {
	select {
	case info.SyncChannel <- struct{}{}:
	default:
	}
}

func (info *RequestInfo) StepDone(state rtCtrlInvokeStage) {
	if info.Input != nil && info.Input.EnableMetrics {
		info.Output.Statistic.Metric.StepDone(state)
	}
}
