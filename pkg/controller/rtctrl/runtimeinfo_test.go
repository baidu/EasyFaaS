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
	"net/url"
	"testing"
	"time"

	"github.com/baidu/openless/pkg/api"
)

func TestRuntimeMap(t *testing.T) {
	rtMap := NewRuntimeManager(getFuncletNode(), &RuntimeManagerParameters{MaxRuntimeIdle: 10, MaxRunnerDefunct: 30})
	rtParams := &NewRuntimeParameters{
		RuntimeID:               "aa",
		ConcurrentMode:          false,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rt := rtMap.NewRuntime(rtParams)
	if rt == nil {
		t.Error("create runtime aa failed.")
	}
	rt2 := rtMap.NewRuntime(rtParams)
	if rt2 != nil {
		t.Error("create runtime with same id")
	}

	rt, err := rtMap.GetRuntime("aa")
	if err != nil {
		t.Errorf("invalid runtime with id %s", "aa")
	}
	rtlist := rtMap.RuntimeList()
	if len(rtlist) == 0 {
		t.Error("runtime list error")
	}
	rtMap.DelRuntime("aa")
}

func TestRuntimeInfo(t *testing.T) {
	rtParams := &NewRuntimeParameters{
		RuntimeID:               "aa",
		ConcurrentMode:          false,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rtInfo := NewRuntimeInfo(rtParams)
	if rtInfo.RuntimeID != "aa" || rtInfo.State != RuntimeStateClosed {
		t.Errorf("runtime mismatch id = %s, status = %s", rtInfo.RuntimeID, rtInfo.State)
	}
	rtInfo.SetInitTime(1, 2)
	rtInfo.SetLoadTime(3, 4)
	if rtInfo.PreInitTimeMS != 1 ||
		rtInfo.PostInitTimeMS != 2 ||
		rtInfo.PreLoadTimeMS != 3 ||
		rtInfo.PostLoadTimeMS != 4 {
		t.Errorf("runtime time mismatch. init[%d-%d], load[%d-%d]",
			rtInfo.PreInitTimeMS, rtInfo.PostInitTimeMS,
			rtInfo.PreLoadTimeMS, rtInfo.PostLoadTimeMS)
	}
}

func TestHandleRuntimeInit(t *testing.T) {
	rtParams := &NewRuntimeParameters{
		RuntimeID:               "aa",
		ConcurrentMode:          false,
		StreamMode:              true,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rt := NewRuntimeInfo(rtParams)
	p := &startRuntimeParams{
		urlParams:  url.Values{},
		warmNotify: make(chan struct{}),
	}

	rt.SetState(RuntimeStateWarmUp)
	p.urlParams.Set("initstart", "10")
	p.urlParams.Set("initdone", "20")
	go func() {
		time.Sleep(1 * time.Second)
		close(rt.runtimeStopChan)
	}()
	err := rt.startRuntimeLoop(p)
	if err != nil ||
		rt.PreInitTimeMS != 10 ||
		rt.PostInitTimeMS != 20 ||
		rt.State != RuntimeStateWarm {
		t.Errorf("handle runtime init failed. %+v; err %s", rt, err)
	}
}

func TestHandleInvokeDone(t *testing.T) {
	rtParams := &NewRuntimeParameters{
		RuntimeID:               "aa",
		ConcurrentMode:          false,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rt := NewRuntimeInfo(rtParams)
	req := NewRequestInfo("bb", rt)

	// runnerinfo:=NewRunnerInfo("aaa")
	rt.requestMap.Store("bb", req)
	params := &url.Values{}
	params.Set("maxmemused", "102400")
	params.Set("success", "true")
	// runnerinfo.maxMemory=5
	ret := rt.handleInvokeDone("bb", params, "result 123")
	if !ret {
		t.Errorf("handle invoke done failed. %+v", req)
	}
}

func TestRuntimeInfo_IsRunnerDefunct(t *testing.T) {
	rtParams := &NewRuntimeParameters{
		RuntimeID:               "aa",
		ConcurrentMode:          false,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rt := NewRuntimeInfo(rtParams)
	// timeout
	now := time.Now()
	deadline := now.Add(-time.Duration(3 * time.Second))
	rt.updateLastLivenessTime()
	res := rt.IsRunnerDefunct(deadline)
	if res != false {
		t.Errorf("runner defunct expected false ,but got %v", res)
	}

	// abnormal
	rt.Abnormal = true
	rt.updateLastLivenessTime()
	deadline = rt.LastLivenessTime.Add(3 * time.Second)
	res = rt.IsRunnerDefunct(deadline)
	if res != true {
		t.Errorf("runner defunct expected true ,but got %v", res)
	}

	// warmup
	rt.Abnormal = false
	rt.updateLastLivenessTime()
	deadline = rt.LastAccessTime.Add(time.Duration(3 * time.Second))
	rt.SetState(RuntimeStateWarmUp)
	res = rt.IsRunnerDefunct(deadline)
	if res != true {
		t.Errorf("runner defunct expected true ,but got %v", res)
	}

	// stopping
	rt.SetState(RuntimeStateStopping)
	res = rt.IsRunnerDefunct(deadline)
	if res != true {
		t.Errorf("runner defunct expected true ,but got %v", res)
	}

	// stopped
	rt.SetState(RuntimeStateStopped)
	res = rt.IsRunnerDefunct(deadline)
	if res != true {
		t.Errorf("runner defunct expected true ,but got %v", res)
	}

	// closed
	rt.SetState(RuntimeStateClosed)
	res = rt.IsRunnerDefunct(deadline)
	if res != true {
		t.Errorf("runner defunct expected true ,but got %v", res)
	}

	// warm
	rt.SetState(RuntimeStateWarm)
	res = rt.IsRunnerDefunct(deadline)
	if res != false {
		t.Errorf("runner defunct expected false ,but got %v", res)
	}
}
