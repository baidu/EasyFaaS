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

package api

import (
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap/zapcore"

	"github.com/baidu/easyfaas/pkg/rest"
)

const (
	RunningModeCommon = "common"
	RunningModeIDE    = "ide"
)

type Event string

const (
	EventInit   Event = "event_init"
	EventWarmup Event = "event_warmup"
	EventReset  Event = "event_reset"
)

const (
	TmpStorageTypeLoop     = "loop"
	TmpStorageTypeHostPath = "host-path"
)

const (
	ContainerIDsParams = "ContainerIDs"
)

type FuncletNodeInfo struct {
	Resource *FuncletResource
}

type FuncletResource struct {
	BaseMemory  uint64
	Default     *Resource
	Capacity    *Resource
	Reserved    *Resource
	Allocatable *Resource
}

func NewFuncletResource() *FuncletResource {
	return &FuncletResource{
		Default:     NewResource(),
		Capacity:    NewResource(),
		Reserved:    NewResource(),
		Allocatable: NewResource(),
	}
}

type Resource struct {
	MilliCPUs int64
	Memory    int64
}

func NewResource() *Resource {
	return &Resource{
		MilliCPUs: 0,
		Memory:    0,
	}
}

func (r *Resource) String() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("mem %d bytes, cpu %d vCore", r.Memory, r.MilliCPUs)
}

func (r *Resource) Copy() *Resource {
	if r == nil {
		return nil
	}
	rp := Resource{}
	rp.Memory = r.Memory
	rp.MilliCPUs = r.MilliCPUs
	return &rp
}

func (r *Resource) Sync(rs *Resource) {
	if rs == nil {
		return
	}
	if r == nil {
		r = &Resource{}
	}
	r.Memory = rs.Memory
	r.MilliCPUs = rs.MilliCPUs
	return
}

type FreezerState string

const (
	Undefined FreezerState = ""
	Frozen    FreezerState = "FROZEN"
	Thawed    FreezerState = "THAWED"
)

// MemoryStats holds the on-demand stastistics from the memory cgroup
type MemoryStats struct {
	// Memory usage (in bytes).
	Usage int64
	// Memory limit (in bytes)
	Limit int64
	// MemorySwap Limit (in bytes)
	SwapLimit int64
}

type CPUStats struct {
	TotalUsage int64
}

// ResourceStats holds on-demand stastistics from various cgroup subsystems
type ResourceStats struct {
	// Memory statistics.
	MemoryStats  *MemoryStats
	CPUStats     *CPUStats
	FreezerState FreezerState
}

type ContainerInfo struct {
	Hostname       string
	ContainerID    string
	HostPid        int
	EventLock      EventLock
	CurrentEvent   Event
	WithStreamMode bool
	IsFrozen       bool
	Resource       *Resource
	ResourceStats  *ResourceStats
}

// MarshalLogObject is marshaler for ContainerInfo
func (c *ContainerInfo) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if c == nil {
		return nil
	}
	enc.AddString("container_id", c.ContainerID)
	enc.AddString("pid", strconv.Itoa(c.HostPid))
	return nil
}

func (c *ContainerInfo) Copy() (nc *ContainerInfo) {
	return &ContainerInfo{
		Hostname:       c.Hostname,
		ContainerID:    c.ContainerID,
		HostPid:        c.HostPid,
		IsFrozen:       c.IsFrozen,
		WithStreamMode: c.WithStreamMode,
		Resource:       c.Resource.Copy(),
	}
}

type EventLock struct {
	c chan struct{}
}

func NewEventLock() EventLock {
	var l EventLock
	l.c = make(chan struct{}, 1)
	l.c <- struct{}{}
	return l
}

func (l EventLock) Lock() bool {
	res := false
	select {
	case <-l.c:
		res = true
	default:
	}
	return res
}

func (l EventLock) UnLock() {
	l.c <- struct{}{}
}

type FuncletClientListNodeInput struct {
	RequestID string
}

type FuncletClientContainerInfoInput struct {
	RequestID string
	ID        string
}

type ContainerInfoResponse = ContainerInfo

type ListContainerCriteria struct {
	rest.QueryCriteria
}

func NewListContainerCriteria() *ListContainerCriteria {
	return &ListContainerCriteria{rest.NewQueryCriteria()}
}
func (c *ListContainerCriteria) AddContainerIDs(IDs []string) {
	IDsStr := strings.Join(IDs, ",")
	c.AddCondition(ContainerIDsParams, IDsStr)
}

func (c *ListContainerCriteria) ReadContainerIDs() (IDs []string, err error) {
	strVal := c.Value().Get(ContainerIDsParams)
	if strVal == "" {
		return nil, nil
	}
	IDs = strings.Split(strVal, ",")
	return IDs, nil
}

type FuncletClientListContainersInput struct {
	Host      string
	Criteria  *ListContainerCriteria
	RequestID string
}

type ListContainersResponse []*ContainerInfo

type FuncletClientWarmUpInput struct {
	ContainerID           string
	RequestID             string
	Code                  *FunctionCodeLocation
	Configuration         *FunctionConfiguration
	RuntimeConfiguration  *RuntimeConfiguration
	NeedScaleUp           bool
	ScaleUpRecommendation *ScaleUpRecommendation
	WithStreamMode        bool // TODO: remove it after apiserver deployed in production
}

type WarmupRequest struct {
	ContainerID string
	RequestID   string
	*WarmUpContainerArgs
}

type WarmUpContainerArgs struct {
	Code                  *CodeStorage
	Configuration         *FunctionConfig
	RuntimeConfiguration  *RuntimeConfiguration
	ScaleUpRecommendation *ScaleUpRecommendation
	WithStreamMode        bool // TODO: remove it after apiserver deployed in production
}

type WarmUpResponse struct {
	Container ContainerInfo
}

type FuncletClientCoolDownInput struct {
	Host                    string
	ContainerID             string
	RequestID               string
	ScaleDownRecommendation *ScaleDownRecommendation
}

type CoolDownRequest struct {
	ContainerID             string
	RequestID               string
	ScaleDownRecommendation *ScaleDownRecommendation
}

type FuncletClientRebornInput struct {
	ContainerID             string
	RequestID               string
	ScaleDownRecommendation *ScaleDownRecommendation
}
type RebornRequest struct {
	ContainerID             string
	RequestID               string
	ScaleDownRecommendation *ScaleDownRecommendation
}

type ResetRequest struct {
	ContainerID             string
	RequestID               string
	ScaleDownRecommendation *ScaleDownRecommendation
}

type ResetResponse struct {
	ScaleDownResult *ScaleDownImplementationResult
}

func NewResetResponse() *ResetResponse {
	return &ResetResponse{
		ScaleDownResult: NewScaleDownImplementationResult(),
	}
}

type ScaleDownImplementationResult struct {
	Success []string
	Fails   map[string]*ContainerInfo
}

func NewScaleDownImplementationResult() *ScaleDownImplementationResult {
	return &ScaleDownImplementationResult{
		Success: make([]string, 0),
		Fails:   make(map[string]*ContainerInfo, 0),
	}
}
