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
	"time"

	"github.com/baidu/openless/pkg/util/logs"
)

//////////////////////////occupy event

type OccupyInput struct {
	CommitID       string
	WithStreamMode bool
	MemorySize     uint64
	MilliCPUs      int64
}

func (info *RuntimeInfo) opOccupyCheck(interface{}) error {
	if info.State != RuntimeStateCold {
		return &RuntimeStateUnmatched{
			RuntimeID:     info.RuntimeID,
			CurrentState:  info.State,
			ExpectedState: []RuntimeStateType{RuntimeStateCold},
		}
	}

	if !info.available() {
		return &RuntimeMatchError{
			Reason: "runner is not available",
		}
	}
	return nil
}

// opOccupySet

func (info *RuntimeInfo) opOccupySet(args interface{}) error {
	params := args.(*OccupyInput)
	info.SetState(RuntimeStateWarmUp)
	info.updateLastAccessTime()
	info.updateStreamMode(params.WithStreamMode)
	info.SetResource(params.MemorySize, params.MilliCPUs)
	info.Concurrency++
	logs.V(5).Infof("occupy runtime %s concurrency %d status %s", info.RuntimeID, info.Concurrency, info.State)
	info.SetCommitID(params.CommitID)
	info.SetMarked(true)
	return nil
}

//////////////////////////merged event

type MergedInput struct {
	CommitID string
}

func (info *RuntimeInfo) opMergedCheck(interface{}) error {
	if info.State != RuntimeStateCold {
		return &RuntimeStateUnmatched{
			RuntimeID:     info.RuntimeID,
			CurrentState:  info.State,
			ExpectedState: []RuntimeStateType{RuntimeStateCold},
		}
	}

	if !info.available() {
		return &RuntimeMatchError{
			Reason: "runner is not available",
		}
	}
	return nil
}

func (info *RuntimeInfo) opMergedSet(args interface{}) error {
	params := args.(*MergedInput)
	info.SetState(RuntimeStateMerged)
	logs.V(5).Infof("runtime merged %s ", info.RuntimeID)
	info.SetCommitID(params.CommitID)
	return nil
}

//////////////////////////merged event

type RetrieveInput struct {
	Deadline time.Time
}

func (info *RuntimeInfo) opRetrieveCheck(args interface{}) error {
	if !(info.State == RuntimeStateMerged || info.State == RuntimeStateReclaiming) {
		return &RuntimeStateUnmatched{
			RuntimeID:     info.RuntimeID,
			CurrentState:  info.State,
			ExpectedState: []RuntimeStateType{RuntimeStateMerged, RuntimeStateReclaiming},
		}
	}
	params := args.(*RetrieveInput)

	if info.State == RuntimeStateReclaiming && info.LastResetTime.After(params.Deadline) {
		return &RuntimeMatchError{
			Reason: "last reset time hasn't timeout",
		}
	}

	if info.Concurrency == 0 {
		return nil
	}

	return &RuntimeMatchError{
		Reason: "runtime info did not match",
	}
}

func (info *RuntimeInfo) opRetrieveSet(interface{}) error {
	info.CommitID = ""
	info.UserID = ""
	info.Concurrency = 0
	info.ConcurrentMode = info.DefaultConcurrentMode
	info.updateLastResetTime()
	info.SetState(RuntimeStateReclaiming)
	return nil
}

/////////////////////////////////////rollback event

type RollbackInput struct {
	CommitID string
}

func (info *RuntimeInfo) opRollbackCheck(args interface{}) error {
	params := args.(*RollbackInput)
	if params.CommitID != info.CommitID {
		return &RuntimeMatchError{
			Reason: "commit id doesn't match",
		}
	}
	if !(info.State == RuntimeStateWarmUp || info.State == RuntimeStateMerged) {
		return &RuntimeStateUnmatched{
			RuntimeID:     info.RuntimeID,
			CurrentState:  info.State,
			ExpectedState: []RuntimeStateType{RuntimeStateWarmUp, RuntimeStateMerged},
		}
	}
	return nil
}

func (info *RuntimeInfo) opRollbackSet(interface{}) error {
	logs.V(5).Infof("occupy runtime %s concurrency %d status %s", info.RuntimeID, info.Concurrency, info.State)
	if info.State == RuntimeStateWarmUp {
		info.Concurrency--
	}
	info.SetState(RuntimeStateCold)
	info.updateStreamMode(false)
	info.SetResource(0, 0)
	info.SetCommitID("")
	info.SetMarked(false)
	return nil
}

/////////////////////////////////////mark event

type MarkInput struct {
	CommitID        string
	ConcurrentQuota uint64
}

func (info *RuntimeInfo) opMarkCheck(args interface{}) error {
	if info.State != RuntimeStateWarmUp && info.State != RuntimeStateWarm {
		return &RuntimeStateUnmatched{
			RuntimeID:     info.RuntimeID,
			CurrentState:  info.State,
			ExpectedState: []RuntimeStateType{RuntimeStateWarmUp, RuntimeStateWarm},
		}
	}

	params := args.(*MarkInput)

	if !info.available() {
		return &RuntimeMatchError{
			Reason: "runner is not available",
		}
	}

	if info.CommitID != params.CommitID {
		return &RuntimeMatchError{
			Reason: "commit id not match",
		}
	}

	if (!info.ConcurrentMode || params.ConcurrentQuota == 0) && info.Concurrency == 0 {
		return nil
	}

	if info.ConcurrentMode && info.Concurrency < params.ConcurrentQuota {
		return nil
	}

	return &RuntimeMatchError{
		Reason: "concurrency exceed limit",
	}
}

// opMarkSet
func (info *RuntimeInfo) opMarkSet(interface{}) error {
	info.updateLastAccessTime()
	info.Concurrency++
	logs.V(6).Infof("mark runtime %s concurrency %d status %s", info.RuntimeID, info.Concurrency, info.State)
	return nil
}

///////////////////////////////stop event

type StopInput struct {
	Deadline time.Time
}

func (info *RuntimeInfo) opStopCheck(args interface{}) error {
	if info.State != RuntimeStateWarm {
		return &RuntimeStateUnmatched{
			RuntimeID:     info.RuntimeID,
			CurrentState:  info.State,
			ExpectedState: []RuntimeStateType{RuntimeStateWarm},
		}
	}

	params := args.(*StopInput)

	if info.Concurrency == 0 && info.LastAccessTime.Before(params.Deadline) {
		return nil
	}

	return &RuntimeMatchError{
		Reason: "no need to stop",
	}
}

// opStopSet
func (info *RuntimeInfo) opStopSet(interface{}) error {
	info.CommitID = ""
	info.UserID = ""
	info.Concurrency = 0
	info.ConcurrentMode = info.DefaultConcurrentMode
	info.SetState(RuntimeStateStopping)
	close(info.runtimeStoppingChan)
	return nil
}

///////////////////////////////reset event

type ResetInput struct {
	Deadline time.Time
}

func (info *RuntimeInfo) opResetCheck(args interface{}) error {
	params := args.(*ResetInput)

	if info.IsRunnerDefunct(params.Deadline) {
		return nil
	}

	return &RuntimeNoNeedToReset{
		RuntimeID: info.RuntimeID,
	}
}

// opStopSet
func (info *RuntimeInfo) opResetSet(interface{}) error {
	info.CommitID = ""
	info.UserID = ""
	info.Concurrency = 0
	info.ConcurrentMode = info.DefaultConcurrentMode
	info.updateLastResetTime()
	return nil
}

// CAS check and set runtime info
func (info *RuntimeInfo) CAS(opType CASOpType, args interface{}) (err error) {
	op := casOps[opType]

	if err = op.check(info, args); err != nil {
		return
	}

	info.invokeLock.Lock()
	defer info.invokeLock.Unlock()

	if err = op.check(info, args); err != nil {
		return
	}

	err = op.set(info, args)
	if err != nil {
		return
	}

	logs.Infof("finished cas %s for runtime %s", op.name, info.RuntimeID)
	return
}

type CASOpType int

const (
	OpOccupy CASOpType = iota
	OpMerged
	OpRetrieve
	OpRollback
	OpMark
	OpInit
	OpStop
	OpReset
	OpClose
	OpEnd
)

type eventCallback func(*RuntimeInfo, interface{}) error

type runtimeEvent struct {
	name  string
	check eventCallback
	set   eventCallback
}

var casOps [OpEnd]*runtimeEvent

func init() {
	casOps[OpOccupy] = &runtimeEvent{
		name:  "occupy",
		check: (*RuntimeInfo).opOccupyCheck,
		set:   (*RuntimeInfo).opOccupySet,
	}

	casOps[OpMerged] = &runtimeEvent{
		name:  "merged",
		check: (*RuntimeInfo).opMergedCheck,
		set:   (*RuntimeInfo).opMergedSet,
	}

	casOps[OpRetrieve] = &runtimeEvent{
		name:  "retrieve",
		check: (*RuntimeInfo).opRetrieveCheck,
		set:   (*RuntimeInfo).opRetrieveSet,
	}

	casOps[OpRollback] = &runtimeEvent{
		name:  "rollback",
		check: (*RuntimeInfo).opRollbackCheck,
		set:   (*RuntimeInfo).opRollbackSet,
	}

	casOps[OpMark] = &runtimeEvent{
		name:  "mark",
		check: (*RuntimeInfo).opMarkCheck,
		set:   (*RuntimeInfo).opMarkSet,
	}

	casOps[OpInit] = &runtimeEvent{
		name:  "init",
		check: nil,
		set:   nil,
	}

	casOps[OpStop] = &runtimeEvent{
		name:  "stop",
		check: (*RuntimeInfo).opStopCheck,
		set:   (*RuntimeInfo).opStopSet,
	}

	casOps[OpReset] = &runtimeEvent{
		name:  "reset",
		check: (*RuntimeInfo).opResetCheck,
		set:   (*RuntimeInfo).opResetSet,
	}

	casOps[OpClose] = &runtimeEvent{
		name:  "close",
		check: nil,
		set:   nil,
	}
}

// Release: release the occupation of runtime
func (info *RuntimeInfo) Release() error {
	info.invokeLock.Lock()
	defer info.invokeLock.Unlock()

	if info.Concurrency == 0 {
		logs.Errorf("can't release runtime %s", info.RuntimeID)
		return &RuntimeReleaseError{
			RuntimeID: info.RuntimeID,
			Reason:    "runtime concurrency has been 0",
		}
	}

	// TODO: what is the reason of updating access time in this situation?
	//info.updateLastAccessTime()
	info.Concurrency--
	logs.V(5).Infof("release runtime %s concurrency %d", info.RuntimeID, info.Concurrency)
	return nil
}
