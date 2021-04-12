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
	"bytes"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/baidu/easyfaas/pkg/util/logs"

	"github.com/baidu/easyfaas/pkg/util/bytefmt"

	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/util/json"
)

type RuntimeDispatcher interface {
	// runtime
	RuntimeList() []*RuntimeInfo
	GetRuntime(string) (*RuntimeInfo, error)
	NewRuntime(*NewRuntimeParameters) *RuntimeInfo
	OccupyColdRuntime(*InvocationInput) (*RuntimeInfo, *api.ScaleUpRecommendation)
	FindWarmRuntime(*InvocationInput) *RuntimeInfo
	CoolDownRuntime(*RuntimeInfo) (*api.ScaleDownRecommendation, error)
	ResetRuntime(*RuntimeInfo) (*api.ScaleDownRecommendation, error)

	// resource
	IncreaseUsedResource(*api.Resource) bool
	ReleaseUsedResource(*api.Resource) bool
	ReleaseMarkedResource(rs *api.Resource) bool
	SyncResource(*api.FuncletResource) bool
	SyncRuntimeResource(string, *api.Resource) (bool, error)

	// statistic
	RuntimeStatistics() (cold, inUse, all int)
	ResourceStatistics() *api.ServiceResource
}

type RuntimeManagerParameters struct {
	// Runtime maximum idle time
	// Units: seconds
	MaxRuntimeIdle int

	// The longest expire time after runner disconnect
	// Units: seconds
	MaxRunnerDefunct int

	// The longest expire time after runner last reset
	// Units: seconds
	MaxRunnerResetTimeout int
}

type occupyColdRuntimeContext struct {
	memBytes *uint64
	input    *OccupyInput
	logger   *logs.Logger
}

type occupyScaleDownContext struct {
	targetRuntime *RuntimeInfo
	memBytes      *uint64
}

type RuntimeManager struct {
	MaxRuntimeIdle        int
	MaxRunnerDefunct      int
	MaxRunnerResetTimeout int
	rtMap                 sync.Map // TODO: No need to add a lock for map
	rtArray               []*RuntimeInfo
	resource              *api.ServiceResource
	resourceLock          sync.RWMutex
}

func NewRuntimeManager(r *api.FuncletNodeInfo, params *RuntimeManagerParameters) *RuntimeManager {
	resource := api.ServiceResource{
		Capacity:    r.Resource.Capacity,
		Allocatable: r.Resource.Allocatable,
		Default:     r.Resource.Default,
		Marked:      api.NewResource(),
		Used:        api.NewResource(),
		BaseMemory:  r.Resource.BaseMemory,
	}
	rtMap := &RuntimeManager{
		MaxRuntimeIdle:        params.MaxRuntimeIdle,
		MaxRunnerDefunct:      params.MaxRunnerDefunct,
		MaxRunnerResetTimeout: params.MaxRunnerResetTimeout,
		rtMap:                 sync.Map{},
		rtArray:               make([]*RuntimeInfo, 0),
		resource:              &resource,
		resourceLock:          sync.RWMutex{},
	}
	return rtMap
}

func (m *RuntimeManager) String() string {
	buf := bytes.NewBuffer(nil)
	rtlist := m.RuntimeList()
	for _, rt := range rtlist {
		b, _ := json.Marshal(rt)
		buf.Write(b)
	}
	return buf.String()
}

func (m *RuntimeManager) RuntimeList() []*RuntimeInfo {
	return m.rtArray
}

func (m *RuntimeManager) NewRuntime(params *NewRuntimeParameters) *RuntimeInfo {
	r := NewRuntimeInfo(params)
	if r.Resource == nil {
		m.resourceLock.RLock()
		defer m.resourceLock.RUnlock()
		r.Resource = m.resource.Default
	}
	_, loaded := m.rtMap.LoadOrStore(params.RuntimeID, r)
	if loaded {
		return nil
	}
	m.rtArray = append(m.rtArray, r)
	return r
}

func (m *RuntimeManager) GetRuntime(runtimeID string) (ri *RuntimeInfo, err error) {
	val, loaded := m.rtMap.Load(runtimeID)
	if !loaded {
		return nil, RuntimeNotExist{RuntimeID: runtimeID}
	}
	rt, ok := val.(*RuntimeInfo)
	if !ok {
		return nil, RuntimeInfoError{RuntimeID: runtimeID}
	}
	return rt, nil
}

func (m *RuntimeManager) RuntimeStatistics() (cold, inUse, all int) {
	rtlist := m.RuntimeList()
	if len(rtlist) == 0 {
		return
	}
	for _, rt := range rtlist {
		switch rt.State {
		case RuntimeStateCold:
			cold++
			all++
		case RuntimeStateWarmUp, RuntimeStateWarm:
			inUse++
			all++
		case RuntimeStateMerged, RuntimeStateReclaiming:
			continue
		default:
			all++
		}
	}
	return
}

func (m *RuntimeManager) ResourceStatistics() (resource *api.ServiceResource) {
	m.resourceLock.RLock()
	defer m.resourceLock.RUnlock()
	resource = m.resource.Copy()
	return
}

func (m *RuntimeManager) DelRuntime(runtimeID string) {
	m.rtMap.Delete(runtimeID)
}

// OccupyColdRuntime
func (m *RuntimeManager) OccupyColdRuntime(req *InvocationInput) (ri *RuntimeInfo, recommend *api.ScaleUpRecommendation) {
	memBytes := functionMemorySizeToBytes(*req.Configuration.MemorySize)
	if !m.checkAndMarkResource(int64(memBytes)) {
		req.Logger.Warnf("resource is insufficient: acquire mem %s, resource %s", memBytes, m.resource)
		return nil, nil
	}
	defer func() {
		if ri == nil {
			markedRes := &api.Resource{Memory: int64(memBytes)}
			req.Logger.Warnf("[resource modify]-[marked]-[decrease]:resource %s reason: occupy runtime failed", markedRes)
			m.ReleaseMarkedResource(markedRes)
		}
	}()
	input := &OccupyInput{
		CommitID:       *req.Configuration.CommitID,
		WithStreamMode: req.WithStreamMode,
		MemorySize:     memBytes,
		MilliCPUs:      m.resource.Default.MilliCPUs,
	}
	ctx := occupyColdRuntimeContext{
		input:    input,
		memBytes: &memBytes,
		logger:   req.Logger,
	}
	needScale := m.isNeedScale(memBytes)

	if !needScale {
		for _, rt := range m.rtArray {
			if err := rt.CAS(OpOccupy, input); err == nil && rt != nil {
				ri = rt
				break
			}
		}
		return ri, nil
	}
	return m.occupyColdWithScaleUpRecommendation(&ctx)
}

func (m *RuntimeManager) occupyColdWithScaleUpRecommendation(ctx *occupyColdRuntimeContext) (ri *RuntimeInfo, recommend *api.ScaleUpRecommendation) {
	var done bool
	recommend = &api.ScaleUpRecommendation{}
	ctx.input.MilliCPUs = m.getMilliCPUsByMemory(ctx.input.MemorySize)
	for _, rt := range m.rtArray {
		if err := rt.CAS(OpOccupy, ctx.input); err == nil && rt != nil {
			ri = rt
			recommend.TargetContainer = rt.RuntimeID
			break
		}
	}
	if ri == nil {
		return nil, nil
	}
	defer func(rt *RuntimeInfo) {
		if !done {
			m.rollbackRuntime(rt, ctx.input.CommitID, ctx.logger)
		}
	}(ri)
	tMem := bytefmt.ByteSize(*ctx.memBytes)
	recommend.TargetMemory = tMem
	scaleCount := int(math.Ceil(float64(*ctx.memBytes/m.resource.BaseMemory)) - 1)
	ctx.logger.V(5).Infof("should occupy %d containers: [mem bytes %d bytes] [base memory %d bytes]",
		scaleCount, ctx.memBytes, m.resource.BaseMemory)

	input := MergedInput{
		CommitID: ctx.input.CommitID,
	}
	var cnt int
	for _, rt := range m.rtArray {
		if cnt == scaleCount {
			break
		}
		if err := rt.CAS(OpMerged, &input); err == nil {
			recommend.MergedContainers = append(recommend.MergedContainers, rt.RuntimeID)
			rtd := rt
			defer func(rtr *RuntimeInfo) {
				if !done {
					m.rollbackRuntime(rtr, ctx.input.CommitID, ctx.logger)
				}
			}(rtd)
			cnt++
		}
	}
	if len(recommend.MergedContainers) == scaleCount {
		done = true
		return ri, recommend
	}
	ctx.logger.Errorf("[scale up runtime %s] should merge %d containers: but merge %d containers: %v",
		ri.RuntimeID, scaleCount, len(recommend.MergedContainers), recommend.MergedContainers)
	return nil, nil
}

func (m *RuntimeManager) CoolDownRuntime(runtime *RuntimeInfo) (recommend *api.ScaleDownRecommendation, err error) {
	deadline := time.Now().Add(-time.Duration(m.MaxRuntimeIdle) * time.Second)
	if err := runtime.CAS(OpStop, &StopInput{Deadline: deadline}); err != nil {
		return nil, err
	}
	ctx := &occupyScaleDownContext{
		targetRuntime: runtime,
	}
	mem := uint64(runtime.Resource.Memory)
	ctx.memBytes = &mem
	needScale := m.isNeedScale(mem)

	if !needScale {
		return nil, nil
	}
	return m.occupyWithScaleDownRecommendation(ctx)
}

func (m *RuntimeManager) ResetRuntime(runtime *RuntimeInfo) (recommend *api.ScaleDownRecommendation, err error) {
	deadline := time.Now().Add(-time.Duration(m.MaxRunnerDefunct) * time.Second)
	if err := runtime.CAS(OpReset, &ResetInput{Deadline: deadline}); err != nil {
		return nil, err
	}
	ctx := &occupyScaleDownContext{
		targetRuntime: runtime,
	}
	mem := uint64(runtime.Resource.Memory)
	ctx.memBytes = &mem
	needScale := m.isNeedScale(mem)

	if !needScale {
		return nil, nil
	}
	return m.occupyWithScaleDownRecommendation(ctx)
}

func (m *RuntimeManager) occupyWithScaleDownRecommendation(ctx *occupyScaleDownContext) (recommend *api.ScaleDownRecommendation, err error) {
	deadline := time.Now().Add(-time.Duration(m.MaxRunnerResetTimeout) * time.Second)
	if ctx.targetRuntime == nil {
		return nil, fmt.Errorf("runtime info couldn't be nil")
	}
	excludeID := ctx.targetRuntime.RuntimeID
	recommend = &api.ScaleDownRecommendation{
		TargetContainer: excludeID,
	}
	scaleCount := int(math.Ceil(float64(*ctx.memBytes/m.resource.BaseMemory)) - 1)
	logs.V(5).Infof("[scale down runtime %s] should reset %d containers: [mem bytes %d bytes] [base memory %d bytes]",
		excludeID, scaleCount, *ctx.memBytes, m.resource.BaseMemory)

	var cnt int
	for _, rt := range m.rtArray {
		if cnt == scaleCount {
			break
		}
		if rt.RuntimeID == excludeID {
			continue
		}
		if err := rt.CAS(OpRetrieve, &RetrieveInput{deadline}); err == nil {
			recommend.ResetContainers = append(recommend.ResetContainers, rt.RuntimeID)
			cnt++
		} else {
			logs.Errorf("retrieve runtime failed: %v", rt)
		}
	}
	if len(recommend.ResetContainers) == scaleCount {
		return recommend, nil
	}
	logs.Errorf("[scale down runtime %s] should reset %d containers: but retrieve %d containers: %v",
		excludeID, scaleCount, len(recommend.ResetContainers), recommend.ResetContainers)
	return nil, nil
}

// FindWarmRuntime
func (m *RuntimeManager) FindWarmRuntime(req *InvocationInput) *RuntimeInfo {
	input := &MarkInput{
		CommitID:        *req.Configuration.CommitID,
		ConcurrentQuota: req.Configuration.PodConcurrentQuota,
	}
	for _, rt := range m.rtArray {
		if err := rt.CAS(OpMark, input); err == nil {
			return rt
		}
	}

	return nil
}

func (m *RuntimeManager) rollbackRuntime(runtime *RuntimeInfo, commitID string, logger *logs.Logger) {
	logger.Warnf("rollback runtime %v commit id %s", runtime, commitID)
	input := RollbackInput{
		CommitID: commitID,
	}
	if err := runtime.CAS(OpRollback, &input); err != nil {
		logger.Errorf("rollback container %s failed: %s", runtime.RuntimeID, err)
		return
	}
	return
}

// checkAndMarkResource
func (m *RuntimeManager) checkAndMarkResource(memBytes int64) bool {
	m.resourceLock.RLock()
	if m.resource.Allocatable.Memory-m.resource.Marked.Memory-memBytes < 0 {
		m.resourceLock.RUnlock()
		return false
	}
	m.resourceLock.RUnlock()

	m.resourceLock.Lock()
	defer m.resourceLock.Unlock()
	m.resource.Marked.Memory += memBytes
	return true
}

// ReleaseUsedResource
func (m *RuntimeManager) IncreaseUsedResource(rs *api.Resource) bool {
	m.resourceLock.Lock()
	defer m.resourceLock.Unlock()
	m.resource.Used.Memory += rs.Memory
	m.resource.Used.MilliCPUs += rs.MilliCPUs
	return true
}

// ReleaseUsedResource
func (m *RuntimeManager) ReleaseUsedResource(rs *api.Resource) bool {
	m.resourceLock.Lock()
	defer m.resourceLock.Unlock()
	m.resource.Used.Memory -= rs.Memory
	m.resource.Used.MilliCPUs -= rs.MilliCPUs
	if m.resource.Used.Memory < 0 || m.resource.Used.MilliCPUs < 0 {
		logs.Errorf("used resource invalid: %s", m.resource)
	}
	return true
}

// ReleaseUsedResource
func (m *RuntimeManager) ReleaseMarkedResource(rs *api.Resource) bool {
	m.resourceLock.Lock()
	defer m.resourceLock.Unlock()
	m.resource.Marked.Memory -= rs.Memory
	if m.resource.Marked.Memory < 0 {
		logs.Errorf("marked resource invalid: %s", m.resource)
	}
	return true
}

func (m *RuntimeManager) SyncResource(resource *api.FuncletResource) bool {
	m.resourceLock.Lock()
	defer m.resourceLock.Unlock()
	if resource.Allocatable.Memory < m.resource.Used.Memory {
		return false
	}
	m.resource.Capacity.Sync(resource.Capacity)
	m.resource.Allocatable.Sync(resource.Allocatable)
	m.resource.Default.Sync(resource.Default)
	m.resource.BaseMemory = resource.BaseMemory
	return true
}

func (m *RuntimeManager) SyncRuntimeResource(ID string, resource *api.Resource) (sync bool, err error) {
	info, err := m.GetRuntime(ID)
	if err != nil {
		return false, RuntimeNotExist{RuntimeID: ID}
	}
	if info.Used || info.Marked {
		return false, RuntimeSyncError{RuntimeID: ID, Reason: "runtime in used"}
	}
	info.Resource.Sync(resource)
	return true, nil
}

func (m *RuntimeManager) isNeedScale(memory uint64) bool {
	return memory > m.resource.BaseMemory
}

func functionMemorySizeToBytes(memSize int64) uint64 {
	return uint64(memSize) * bytefmt.Megabyte
}

func (m *RuntimeManager) getMilliCPUsByMemory(memory uint64) int64 {
	return int64(memory/m.resource.BaseMemory) * m.resource.Default.MilliCPUs
}
