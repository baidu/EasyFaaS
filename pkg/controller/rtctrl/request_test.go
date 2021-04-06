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
	"testing"
	"time"

	"github.com/baidu/openless/pkg/api"
)

func TestRequestInfo(t *testing.T) {
	rtParams := &NewRuntimeParameters{
		RuntimeID:               "bb",
		ConcurrentMode:          false,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rt := NewRuntimeInfo(rtParams)
	req := NewRequestInfo("aa", rt)
	req.SetInitTime(1, 2)
	if req.InitStartTimeMS != 1 ||
		req.InitDoneTimeMS != 2 {
		t.Errorf("init time error: [%d-%d]", req.InitStartTimeMS, req.InitDoneTimeMS)
	}

	req.InvokeResult(StatusSuccess, "result")
}

func TestInvokeDone(t *testing.T) {
	rtParams := &NewRuntimeParameters{
		RuntimeID:               "bb",
		ConcurrentMode:          false,
		StreamMode:              false,
		WaitRuntimeAliveTimeout: 3,
		Resource:                &api.Resource{},
	}
	rt := NewRuntimeInfo(rtParams)
	req := NewRequestInfo("aa", rt)
	nowT := time.Now()
	startT := nowT.UnixNano()
	endT := nowT.Add(time.Millisecond + 100*time.Microsecond).UnixNano()
	req.SetInitTime(startT, endT)
	//req.store =
	req.InvokeDone()
	t.Logf("reqinfo = %+v", req)
}
