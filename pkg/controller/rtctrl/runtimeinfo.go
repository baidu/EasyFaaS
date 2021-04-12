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
	"sync"
	"time"

	"github.com/baidu/easyfaas/pkg/util/logs"
)

func NewRuntimeInfo(params *NewRuntimeParameters) *RuntimeInfo {
	ri := &RuntimeInfo{
		RuntimeID:               params.RuntimeID,
		LastAccessTime:          time.Now(),
		LastLivenessTime:        time.Now(),
		ConcurrentMode:          params.ConcurrentMode,
		DefaultConcurrentMode:   params.ConcurrentMode,
		Abnormal:                false,
		State:                   RuntimeStateClosed,
		runtimeRunChan:          make(chan struct{}),
		runtimeStopChan:         make(chan struct{}),
		runtimeStoppingChan:     make(chan struct{}),
		requestChan:             make(chan *InvokeRequest, 100), // TODO: length is a magic num
		httpClient:              nil,
		runtimeWaitGroup:        sync.WaitGroup{},
		WaitRuntimeAliveTimeout: params.WaitRuntimeAliveTimeout,
		WithStreamMode:          params.StreamMode,
		Resource:                params.Resource,
	}
	if params.IsFrozen {
		ri.SetState(RuntimeStateMerged)
	}
	return ri
}

func (info *RuntimeInfo) SetState(s RuntimeStateType) {
	logs.V(3).Infof("update runtime %s state %s to %s", info.RuntimeID, info.State, s)
	info.State = s
}

func (info *RuntimeInfo) SetResource(mem uint64, cpu int64) {
	logs.V(3).Infof("update runtime %s memory size %d to %d; cpu size %d to %d", info.RuntimeID,
		info.Resource.Memory, mem, info.Resource.MilliCPUs, cpu)
	info.Resource.Memory = int64(mem)
	info.Resource.MilliCPUs = cpu
}

func (info *RuntimeInfo) SetMemorySize(mem uint64) {
	logs.V(3).Infof("update runtime %s memory size %d to %d", info.RuntimeID, info.MemorySize, mem)
	info.MemorySize = mem
}

func (info *RuntimeInfo) SetLoadTime(pre, post int64) {
	info.PreLoadTimeMS = pre
	info.PostLoadTimeMS = post
}

func (info *RuntimeInfo) SetInitTime(pre, post int64) {
	info.PreInitTimeMS = pre
	info.PostInitTimeMS = post
}

func (info *RuntimeInfo) SetCommitID(cm string) {
	info.CommitID = cm
}

// RebootBegin
func (info *RuntimeInfo) RebootBegin() {
	info.rebootWaitGroup.Add(1)
}

// RebootEnd
func (info *RuntimeInfo) RebootEnd() {
	info.rebootWaitGroup.Done()
}

func (info *RuntimeInfo) RebootWait() {
	info.rebootLock.Lock()
	defer info.rebootLock.Unlock()
	info.rebootWaitGroup.Wait()
}

func (info *RuntimeInfo) SetMarked(m bool) {
	info.Marked = m
}

func (info *RuntimeInfo) SetUsed(m bool) {
	info.Used = m
}

func (info *RuntimeInfo) Invalidate() {
	info.invokeLock.Lock()
	defer info.invokeLock.Unlock()

	logs.Infof("invalidate runtime %s", info.RuntimeID)

	if info.Abnormal {
		return
	}

	info.Abnormal = true
	info.AbnormalTimes++
}

// updateLastAccessTime
func (info *RuntimeInfo) updateLastAccessTime() {
	info.LastAccessTime = time.Now()
}

// updateLastResetTime
func (info *RuntimeInfo) updateLastResetTime() {
	info.LastResetTime = time.Now()
}

// updateLastLivenessTime
func (info *RuntimeInfo) updateLastLivenessTime() {
	info.LastLivenessTime = time.Now()
}

// available
func (info *RuntimeInfo) available() bool {
	return !info.Abnormal
}

// updateStreamMode: update runtime stream mode
func (info *RuntimeInfo) updateStreamMode(mode bool) {
	info.WithStreamMode = mode
}

// IsRunnerDefunct
func (info *RuntimeInfo) IsRunnerDefunct(deadline time.Time) bool {
	if !info.LastLivenessTime.Before(deadline) {
		return false
	}

	switch {
	case info.Abnormal:
		return true

	case info.State == RuntimeStateWarmUp:
		return true

	case info.State == RuntimeStateStopping:
		return true

	case info.State == RuntimeStateStopped:
		return true

	case info.State == RuntimeStateClosed:
		return true
	}

	return false
}
