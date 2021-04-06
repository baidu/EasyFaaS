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
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/baidu/openless/pkg/util/bytefmt"

	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/service/lambda"

	"github.com/baidu/openless/pkg/api"
)

var minMemory = api.MinMemorySize

func getFuncletNode() *api.FuncletNodeInfo {
	return &api.FuncletNodeInfo{
		Resource: &api.FuncletResource{
			Capacity: &api.Resource{
				Memory:    1342177280000,
				MilliCPUs: 5000,
			},
			Allocatable: &api.Resource{
				Memory:    1342177280000,
				MilliCPUs: 5000,
			},
			Default: &api.Resource{
				Memory:    134217728,
				MilliCPUs: 100,
			},
			BaseMemory: 134217728,
		},
	}
}

func initRuntimeList(num int) *RuntimeManager {
	rtMap := NewRuntimeManager(getFuncletNode(), &RuntimeManagerParameters{MaxRuntimeIdle: 10, MaxRunnerDefunct: 30})
	// prepare a runtime list
	for i := 0; i < num; i++ {
		rtParams := &NewRuntimeParameters{
			RuntimeID:               "runtime-" + strconv.Itoa(i),
			ConcurrentMode:          true,
			StreamMode:              false,
			WaitRuntimeAliveTimeout: 3,
			Resource: &api.Resource{
				MilliCPUs: 100,
				Memory:    134217728,
			},
		}
		rt := rtMap.NewRuntime(rtParams)
		rt.SetState(RuntimeStateCold)
	}
	return rtMap
}

func TestGetRuntime(t *testing.T) {
	rtMap := initRuntimeList(1)
	id := "runtime-id-001"
	rtMap.rtMap.Store(id, "xxx")
	_, err := rtMap.GetRuntime(id)
	assert.Equal(t, RuntimeInfoError{id}.Error(), err.Error())
}

func TestFindWarmRuntime(t *testing.T) {
	rtMap := initRuntimeList(3)

	// warm up one
	cmID := "commitID-test"
	input := &InvocationInput{
		Configuration: &api.FunctionConfiguration{
			CommitID:           &cmID,
			PodConcurrentQuota: 4,
			FunctionConfiguration: lambda.FunctionConfiguration{
				MemorySize: &minMemory,
			},
		},
		WithStreamMode: false,
	}
	rtinfo, _ := rtMap.OccupyColdRuntime(input)
	assert.NotEqual(t, rtinfo, nil)

	rtID := rtinfo.RuntimeID
	rtinfo.Release()

	// find a warm one
	info := rtMap.FindWarmRuntime(input)
	// occupy the runtime, not release until testing end
	defer info.Release()
	if info.RuntimeID != rtID {
		t.Errorf("runtime id should be  %s, but got %s", rtID, info.RuntimeID)
	}

	// try to occupy a runtime with bigger memory size
	cmID3 := "commitID-test-3"
	mem := minMemory * 2
	input3 := &InvocationInput{
		Configuration: &api.FunctionConfiguration{
			CommitID:           &cmID3,
			PodConcurrentQuota: 1,
			FunctionConfiguration: lambda.FunctionConfiguration{
				MemorySize: &mem,
			},
		},
		WithStreamMode: false,
	}
	rtinfo3, recommend := rtMap.OccupyColdRuntime(input3)
	assert.NotEqual(t, rtinfo3, nil)
	assert.NotEqual(t, recommend, nil)

	// try to occupy a runtime with another function info
	cmID2 := "commitID-test-2"
	input2 := &InvocationInput{
		Configuration: &api.FunctionConfiguration{
			CommitID:           &cmID2,
			PodConcurrentQuota: 1,
			FunctionConfiguration: lambda.FunctionConfiguration{
				MemorySize: &minMemory,
			},
		},
		WithStreamMode: true,
	}
	rtinfo2, _ := rtMap.OccupyColdRuntime(input2)
	assert.NotEqual(t, rtinfo2, nil)

	info2 := rtMap.FindWarmRuntime(input2)
	assert.NotEqual(t, info2, nil)
	return
}

func TestRuntimeStatistics(t *testing.T) {
	totalNum := 5
	rtMap := initRuntimeList(totalNum)
	rtMapList := rtMap.RuntimeList()
	t.Logf("runtime list len %d", len(rtMapList))
	cmID := "xxx"
	req := &InvocationInput{
		Configuration: &api.FunctionConfiguration{
			CommitID:           &cmID,
			PodConcurrentQuota: 4,
			FunctionConfiguration: lambda.FunctionConfiguration{
				MemorySize: &minMemory,
			},
		},
		WithStreamMode: false,
	}
	rtMap.OccupyColdRuntime(req)
	cold, inuse, all := rtMap.RuntimeStatistics()
	assert.Equal(t, cold, totalNum-1)
	assert.Equal(t, inuse, 1)
	assert.Equal(t, all, totalNum)
}

func TestCoolDownRuntime(t *testing.T) {
	rtMap := initRuntimeList(2)

	// warm up one
	cmID := "commitID-test"
	mem := minMemory * 2
	input := &InvocationInput{
		Configuration: &api.FunctionConfiguration{
			CommitID:           &cmID,
			PodConcurrentQuota: 4,
			FunctionConfiguration: lambda.FunctionConfiguration{
				MemorySize: &mem,
			},
		},
		WithStreamMode: false,
	}
	rtinfo, _ := rtMap.OccupyColdRuntime(input)
	assert.NotEqual(t, rtinfo, nil)

	rtinfo.SetState(RuntimeStateWarm)
	rtinfo.Release()
	_, err := rtMap.CoolDownRuntime(rtinfo)
	assert.Equal(t, err.Error(), RuntimeMatchError{Reason: "no need to stop"}.Error())

	rtinfo.LastAccessTime = time.Now().Add(-time.Duration(11) * time.Second)
	recommend, err2 := rtMap.CoolDownRuntime(rtinfo)
	assert.Equal(t, err2, nil)
	assert.NotEqual(t, recommend, nil)
	return
}

func TestResetRuntime(t *testing.T) {
	rtMap := initRuntimeList(2)

	// warm up one
	cmID := "commitID-test"
	mem := minMemory * 2
	input := &InvocationInput{
		Configuration: &api.FunctionConfiguration{
			CommitID:           &cmID,
			PodConcurrentQuota: 4,
			FunctionConfiguration: lambda.FunctionConfiguration{
				MemorySize: &mem,
			},
		},
		WithStreamMode: false,
	}
	rtinfo, _ := rtMap.OccupyColdRuntime(input)
	if rtinfo == nil {
		t.Errorf("should occupy one cold runtime, but got nil")
		return
	}
	rtinfo.Release()

	_, err := rtMap.ResetRuntime(rtinfo)
	assert.Equal(t, err.Error(), RuntimeNoNeedToReset{RuntimeID: rtinfo.RuntimeID}.Error())
	rtinfo.LastLivenessTime = time.Now().Add(-time.Duration(31) * time.Second)
	recommend, err2 := rtMap.ResetRuntime(rtinfo)
	t.Logf("err %v", err2)
	assert.Equal(t, err2, nil)
	assert.NotEqual(t, recommend, nil)
	return
}

func TestResource(t *testing.T) {
	rtMap := initRuntimeList(10)
	ors := rtMap.ResourceStatistics()
	t.Logf("resource %+v", ors)
	cmID := "commitID-test"
	mem := minMemory * 8
	input := &InvocationInput{
		Configuration: &api.FunctionConfiguration{
			CommitID:           &cmID,
			PodConcurrentQuota: 4,
			FunctionConfiguration: lambda.FunctionConfiguration{
				MemorySize: &mem,
			},
		},
		WithStreamMode: false,
	}
	rtinfo, _ := rtMap.OccupyColdRuntime(input)
	assert.NotEqual(t, rtinfo, nil)

	rs := rtMap.ResourceStatistics()
	assert.Equal(t, mem*bytefmt.Megabyte, rs.Marked.Memory-ors.Marked.Memory)
	t.Logf("rs = %s", rtinfo.Resource)
	rtMap.ReleaseMarkedResource(rtinfo.Resource.Copy())
	rs2 := rtMap.ResourceStatistics()
	assert.Equal(t, mem*bytefmt.Megabyte, rs.Marked.Memory-rs2.Marked.Memory)
}

func TestRuntimeDelete(t *testing.T) {
	rtMap := initRuntimeList(1)
	rtMapList := rtMap.RuntimeList()
	t.Logf("runtime list len %d", len(rtMapList))
	id := rtMapList[0].RuntimeID
	rtMap.DelRuntime(id)
	t.Logf("runtime list %s", rtMap.String())
	info, err := rtMap.GetRuntime(id)
	assert.NotEqual(t, info, nil)
	assert.Equal(t, err.Error(), RuntimeNotExist{RuntimeID: id}.Error())

}

type benchmarkRuntimeArgs struct {
	Concurrency   uint64
	FunctionCount int
	RuntimeCount  int
}

func findAndMarkRuntimeMap(b *testing.B, args *benchmarkRuntimeArgs) {
	rtMap := NewRuntimeManager(getFuncletNode(), &RuntimeManagerParameters{MaxRuntimeIdle: 10, MaxRunnerDefunct: 30})
	for i := 0; i < args.RuntimeCount; i++ {
		rtParams := &NewRuntimeParameters{
			RuntimeID:               "runtime" + strconv.Itoa(i),
			ConcurrentMode:          true,
			StreamMode:              false,
			WaitRuntimeAliveTimeout: 3,
			Resource:                &api.Resource{},
		}
		rt := rtMap.NewRuntime(rtParams)

		params := &startRunnerParams{}
		rt.initRunner(params)
		str := "commitID" + strconv.Itoa(i%args.FunctionCount)
		input := &InvocationInput{
			Configuration: &api.FunctionConfiguration{
				CommitID: &str,
				FunctionConfiguration: lambda.FunctionConfiguration{
					MemorySize: &minMemory,
				},
			},
			WithStreamMode: false,
		}

		rtMap.OccupyColdRuntime(input)
		rt.Release()
	}

	maxParallel := int(args.Concurrency) * args.RuntimeCount
	p := maxParallel / runtime.GOMAXPROCS(0)
	b.Log("Parallelism ", p*runtime.GOMAXPROCS(0))
	b.SetParallelism(p)
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			commitID := "commitID" + strconv.Itoa(i%args.FunctionCount)
			rt := rtMap.FindWarmRuntime(&InvocationInput{
				Configuration: &api.FunctionConfiguration{
					CommitID:           &commitID,
					PodConcurrentQuota: args.Concurrency,
				},
			})
			if rt != nil {
				rt.Release()
			} else {
				b.Error("empty runtime", i)
			}
		}
	})
}

func BenchmarkFindAndMark_1Conc1Func10Rt(b *testing.B) {
	findAndMarkRuntimeMap(b, &benchmarkRuntimeArgs{
		Concurrency:   1,
		FunctionCount: 1,
		RuntimeCount:  10,
	})
}

func BenchmarkFindAndMark_1Conc1Func100Rt(b *testing.B) {
	findAndMarkRuntimeMap(b, &benchmarkRuntimeArgs{
		Concurrency:   1,
		FunctionCount: 1,
		RuntimeCount:  100,
	})
}

func BenchmarkFindAndMark_1Conc1Func400Rt(b *testing.B) {
	findAndMarkRuntimeMap(b, &benchmarkRuntimeArgs{
		Concurrency:   1,
		FunctionCount: 1,
		RuntimeCount:  400,
	})
}

func BenchmarkFindAndMark_1Conc1Func800Rt(b *testing.B) {
	findAndMarkRuntimeMap(b, &benchmarkRuntimeArgs{
		Concurrency:   1,
		FunctionCount: 1,
		RuntimeCount:  800,
	})
}

func BenchmarkFindAndMark_1Conc10Func800Rt(b *testing.B) {
	findAndMarkRuntimeMap(b, &benchmarkRuntimeArgs{
		Concurrency:   1,
		FunctionCount: 50,
		RuntimeCount:  500,
	})
}

func BenchmarkFindAndMark_100Conc1Func10Rt(b *testing.B) {
	findAndMarkRuntimeMap(b, &benchmarkRuntimeArgs{
		Concurrency:   100,
		FunctionCount: 1,
		RuntimeCount:  10,
	})
}

func BenchmarkFindAndMark_200Conc1Func10Rt(b *testing.B) {
	findAndMarkRuntimeMap(b, &benchmarkRuntimeArgs{
		Concurrency:   200,
		FunctionCount: 1,
		RuntimeCount:  10,
	})
}

func BenchmarkFindAndMark_400Conc1Func10Rt(b *testing.B) {
	findAndMarkRuntimeMap(b, &benchmarkRuntimeArgs{
		Concurrency:   400,
		FunctionCount: 1,
		RuntimeCount:  10,
	})
}
