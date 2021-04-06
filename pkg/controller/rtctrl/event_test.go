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
	"testing"
	"time"

	"github.com/baidu/openless/pkg/util/bytefmt"
)

func TestEventOccupy(t *testing.T) {
	rtMap := initRuntimeList(1)
	list := rtMap.RuntimeList()
	rt := list[0]
	rt.SetState(RuntimeStateClosed)
	err := rt.CAS(OpOccupy, &OccupyInput{CommitID: "xxx", WithStreamMode: false})
	expectedErr := &RuntimeStateUnmatched{RuntimeID: rt.RuntimeID, CurrentState: RuntimeStateClosed, ExpectedState: []string{RuntimeStateCold}}
	if err.Error() != expectedErr.Error() {
		t.Errorf("occupy runtime expected %s, but got %s", expectedErr, err)
		return
	}

	rt.SetState(RuntimeStateCold)
	rt.Invalidate()
	err1 := rt.CAS(OpOccupy, &OccupyInput{
		CommitID:       "xxx",
		WithStreamMode: false,
		MemorySize:     uint64(128 * bytefmt.Megabyte),
	})
	expectedErr1 := &RuntimeMatchError{
		Reason: "runner is not available",
	}
	if err1.Error() != expectedErr1.Error() {
		t.Errorf("occupy runtime expected %s, but got %s", expectedErr, err)
		return
	}
}

func TestEventMerged(t *testing.T) {
	rtMap := initRuntimeList(1)
	list := rtMap.RuntimeList()
	rt := list[0]
	rt.SetState(RuntimeStateClosed)
	err := rt.CAS(OpMerged, &MergedInput{CommitID: "xxx"})
	expectedErr := &RuntimeStateUnmatched{RuntimeID: rt.RuntimeID, CurrentState: RuntimeStateClosed, ExpectedState: []string{RuntimeStateCold}}
	if err.Error() != expectedErr.Error() {
		t.Errorf("merge runtime expected %s, but got %s", expectedErr, err)
		return
	}

	rt.SetState(RuntimeStateCold)
	rt.Invalidate()
	err1 := rt.CAS(OpMerged, &MergedInput{CommitID: "xxx"})
	expectedErr1 := &RuntimeMatchError{
		Reason: "runner is not available",
	}
	if err1.Error() != expectedErr1.Error() {
		t.Errorf("merge runtime expected %s, but got %s", expectedErr, err)
		return
	}

	rt.SetState(RuntimeStateCold)
	rt.Abnormal = false
	err2 := rt.CAS(OpMerged, &MergedInput{CommitID: "xxx"})
	if err2 != nil {
		t.Errorf("merge runtime expected nil, but got %s", err2)
		return
	}
}

func TestEventMark(t *testing.T) {
	rtMap := initRuntimeList(1)
	list := rtMap.RuntimeList()
	rt := list[0]
	err := rt.CAS(OpMark, &MarkInput{CommitID: "xxx", ConcurrentQuota: 1})
	expectedErr := &RuntimeStateUnmatched{
		RuntimeID:     rt.RuntimeID,
		CurrentState:  RuntimeStateCold,
		ExpectedState: []string{RuntimeStateWarmUp, RuntimeStateWarm},
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("mark runtime expected %s, but got %s", expectedErr, err)
		return
	}

	rt.CAS(OpOccupy, &OccupyInput{CommitID: "xxx", WithStreamMode: false})
	err2 := rt.CAS(OpMark, &MarkInput{CommitID: "xxx", ConcurrentQuota: 1})
	expectedErr2 := &RuntimeMatchError{
		Reason: "concurrency exceed limit",
	}
	if err2.Error() != expectedErr2.Error() {
		t.Errorf("mark runtime expected %s, but got %s", expectedErr2, err2)
		return
	}

	rt.SetState(RuntimeStateWarm)
	rt.Invalidate()
	err1 := rt.CAS(OpMark, &MarkInput{CommitID: "xxx", ConcurrentQuota: 1})
	expectedErr1 := &RuntimeMatchError{
		Reason: "runner is not available",
	}
	if err1.Error() != expectedErr1.Error() {
		t.Errorf("mark runtime expected %s, but got %s", expectedErr1, err1)
		return
	}
}

func TestEventStop(t *testing.T) {
	rtMap := initRuntimeList(1)
	list := rtMap.RuntimeList()
	rt := list[0]

	// success
	rt.SetState(RuntimeStateWarm)
	rt.updateLastAccessTime()
	idleTime := time.Second
	deadline := time.Now().Add(-idleTime)
	err := rt.CAS(OpStop, &StopInput{Deadline: deadline})
	expectedErr := &RuntimeMatchError{
		Reason: "no need to stop",
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("runtime state expected %s, but got %s", expectedErr, err)
	}
	<-time.NewTicker(idleTime + 1).C
	deadline = time.Now().Add(idleTime)
	err1 := rt.CAS(OpStop, &StopInput{Deadline: deadline})
	if err1 != nil {
		t.Errorf("stop runtime failed %s", err1)
		return
	}
	if rt.State != RuntimeStateStopping {
		t.Errorf("runtime state expected stopping, but got %s", rt.State)
		return
	}

	// fail
	rt.SetState(RuntimeStateClosed)
	err2 := rt.CAS(OpStop, &StopInput{Deadline: deadline})
	expectedErr2 := &RuntimeStateUnmatched{
		RuntimeID:     rt.RuntimeID,
		CurrentState:  RuntimeStateClosed,
		ExpectedState: []string{RuntimeStateWarm},
	}
	if err2.Error() != expectedErr2.Error() {
		t.Errorf("stop runtime expected %s, but got %s", expectedErr2, err2)
		return
	}
}

func TestRelease(t *testing.T) {
	rtMap := initRuntimeList(1)
	list := rtMap.RuntimeList()
	rt := list[0]
	err := rt.Release()
	expectedErr := &RuntimeReleaseError{
		RuntimeID: rt.RuntimeID,
		Reason:    "runtime concurrency has been 0",
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("release runtime expected %s, but got %s", expectedErr, err)
		return
	}
}
