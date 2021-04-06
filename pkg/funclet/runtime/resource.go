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

// Package runtime
package runtime

import (
	"fmt"
	"math"

	"github.com/baidu/openless/pkg/util/logs"

	runtimeErr "github.com/baidu/openless/pkg/funclet/runtime/error"

	"github.com/baidu/openless/pkg/api"

	runtimeapi "github.com/baidu/openless/pkg/funclet/runtime/api"

	"github.com/baidu/openless/pkg/funclet/runtime/cgroup"
	"github.com/baidu/openless/pkg/util/bytefmt"
)

type ResourceManager interface {
	// HasSufficientResources
	HasSufficientResources(o *ResourceOption, containerNum int) (has bool, err error)
	// GetDefaultResourceConfig
	GetDefaultResourceConfig() *runtimeapi.ResourceConfig
	// GetResourceConfigByReadableMemory
	GetResourceConfigByReadableMemory(memoryStr string) (config *runtimeapi.ResourceConfig, err error)
	// GetTotalResources
	GetTotalResources() *api.FuncletResource
	// FrozenContainer
	FrozenContainer(ID string) error
	// ThawContainer
	ThawContainer(ID string) error
	// ContainerResources
	ContainerResources(ID string) (resource *api.Resource, err error)
	// ContainerResourceStats
	ContainerResourceStats(ID string) (stats *api.ResourceStats, err error)
	// UpdateContainerResource
	UpdateContainerResource(ID string, config *runtimeapi.ResourceConfig) error
}

type ResourceControl struct {
	Options               *ResourceOption
	baseMem               uint64
	totalResource         *api.FuncletResource
	baseResourceParams    *cgroup.ResourceParams
	defaultResourceConfig *runtimeapi.ResourceConfig
	cgroupManager         cgroup.CgroupManager
}

func NewResourceManager(o *ResourceOption, containerNum int) (rm ResourceManager, err error) {
	cs, err := cgroup.GetCgroupSubsystems()
	if err != nil {
		return nil, err
	}

	cm, err := cgroup.NewCgroupManager(cs, o.CgroupRootPath, "cgroupfs")
	if err != nil {
		return nil, err
	}
	rc := ResourceControl{
		Options:       o,
		cgroupManager: cm,
	}
	if has, err := rc.HasSufficientResources(o, containerNum); !has || err != nil {
		return nil, runtimeErr.ErrInsufficientResources{Has: has, Err: err}
	}
	return &rc, nil
}

// HasSufficientResources check the total resource is sufficient
func (rc *ResourceControl) HasSufficientResources(o *ResourceOption, containerNum int) (has bool, err error) {
	rp := cgroup.ResourceParams{MemLimits: o.MemLimits, CPURequests: o.CPURequests, CPULimits: o.CPULimits}
	config, err := rc.cgroupManager.GetResourcesConfig(&rp)
	if err != nil {
		return false, err
	}

	rc.totalResource = api.NewFuncletResource()

	// memory capacity
	rc.totalResource.Capacity.Memory, err = rc.getMemoryCapacity(o.TotalMemory)
	if err != nil {
		return false, err
	}
	logs.Debugf("total resource capacity memory:  %d", rc.totalResource.Capacity.Memory)

	// cpu capacity
	rc.totalResource.Capacity.MilliCPUs, err = rc.getMilliCPUCapacity(o.TotalCPUs)
	if err != nil {
		return false, err
	}
	logs.Debugf("total resource capacity cpu: %d", rc.totalResource.Capacity.MilliCPUs)

	// reserved memory
	reservedMemoryUint, err := bytefmt.ToBytes(o.ReservedMemory)
	if err != nil {
		return false, err
	}
	reservedMemory := int64(reservedMemoryUint)
	if reservedMemory >= rc.totalResource.Capacity.Memory {
		return false, fmt.Errorf("reserved memory configuration invalid")
	}
	rc.totalResource.Reserved.Memory = reservedMemory

	// reserved cpu
	reservedMilliCPUs := int64(o.ReservedCPUs * 1000)
	if reservedMilliCPUs >= rc.totalResource.Capacity.MilliCPUs {
		return false, fmt.Errorf("reserved cpu configuration invalid")
	}
	rc.totalResource.Reserved.MilliCPUs = reservedMilliCPUs

	// available memory = total memory - reserved memory
	rc.totalResource.Allocatable.Memory = rc.totalResource.Capacity.Memory - rc.totalResource.Reserved.Memory

	// available cpu = total cpu - reserved cpu
	rc.totalResource.Allocatable.MilliCPUs = rc.totalResource.Capacity.MilliCPUs - rc.totalResource.Reserved.MilliCPUs

	// check sufficient memory
	if *config.Memory*int64(containerNum) > rc.totalResource.Allocatable.Memory {
		return false, nil
	}
	rc.totalResource.BaseMemory = uint64(*config.Memory)
	rc.totalResource.Default.Memory = *config.Memory
	// check sufficient cpu
	containerMilliCPUs := int64(o.CPULimits * 1000)
	rc.totalResource.Default.MilliCPUs = containerMilliCPUs
	if containerMilliCPUs > rc.totalResource.Allocatable.MilliCPUs {
		return false, nil
	}

	rc.defaultResourceConfig = config
	rc.baseResourceParams = &rp
	rc.baseMem = uint64(*rc.defaultResourceConfig.Memory)
	return true, nil
}

func (rc *ResourceControl) GetTotalResources() *api.FuncletResource {
	return rc.totalResource
}

func (rc *ResourceControl) GetDefaultResourceConfig() *runtimeapi.ResourceConfig {
	return rc.defaultResourceConfig
}

func (rc *ResourceControl) GetResourceConfigByReadableMemory(memoryStr string) (config *runtimeapi.ResourceConfig, err error) {
	targetMem, err := bytefmt.ToBytes(memoryStr)
	base := math.Ceil(float64(targetMem / rc.baseMem))
	rp := cgroup.ResourceParams{
		MemLimits:   memoryStr,
		CPURequests: base * rc.baseResourceParams.CPURequests,
		CPULimits:   base * rc.baseResourceParams.CPULimits,
	}
	if int64(rp.CPULimits*1000) > rc.totalResource.Allocatable.MilliCPUs {
		rp.CPULimits = float64(rc.totalResource.Allocatable.MilliCPUs / 1000)
	}
	if int64(rp.CPURequests*1000) > rc.totalResource.Allocatable.MilliCPUs {
		rp.CPURequests = float64(rc.totalResource.Allocatable.MilliCPUs / 1000)
	}
	logs.Debugf("[GetResourceConfigByReadableMemory] mem %s, config params %+v", memoryStr, rp)
	return rc.cgroupManager.GetResourcesConfig(&rp)
}

func (rc *ResourceControl) getMemoryCapacity(totalMemory string) (mem int64, err error) {
	// memory capacity
	// parse the configuration of total memory
	var memoryQuotaConfiguration uint64
	if totalMemory != "" {
		memoryQuotaConfiguration, err = bytefmt.ToBytes(totalMemory)
		if err != nil {
			return 0, err
		}
	}

	// parse memory limits from the cgroup
	cgroupMemoryPath, err := rc.cgroupManager.GetCgroupSubsysPath("memory")
	if err != nil {
		return 0, err
	}
	totalMemoryQuota, err := cgroup.GetCgroupParamUint(cgroupMemoryPath, cgroup.MemoryLimitsFile)
	if err != nil {
		return 0, err
	}

	// the final total memory equals to the configuration of total memory, when memory cgroup had no limits.
	if totalMemoryQuota == cgroup.MemoryNoLimits && memoryQuotaConfiguration != 0 {
		totalMemoryQuota = memoryQuotaConfiguration
	}
	return int64(totalMemoryQuota), nil
}

func (rc *ResourceControl) getMilliCPUCapacity(totalCPU float64) (cpu int64, err error) {
	// cpu capacity
	// parse cpu period from the cgroup
	cgroupCPUPath, err := rc.cgroupManager.GetCgroupSubsysPath("cpu")
	if err != nil {
		return 0, err
	}
	CPUPeriodUint, err := cgroup.GetCgroupParamUint(cgroupCPUPath, cgroup.CFSPeriodFile)
	if err != nil {
		return 0, err
	}
	CPUPeriod := int64(CPUPeriodUint)

	// parse the configuration of total memory
	var milliCPUsConfiguration, cpuQuotaConfiguration int64
	if totalCPU != 0 {
		milliCPUsConfiguration = int64(totalCPU * 1000)
		cpuQuotaConfiguration = cgroup.MilliCPUToQuota(milliCPUsConfiguration, CPUPeriod)
	}
	logs.Debugf("milli cpu from configuration: %d", milliCPUsConfiguration)
	logs.Debugf("cpu quota from configuration: %d", cpuQuotaConfiguration)
	// parse cpu quota from the cgroup
	totalCPUQuota, err := cgroup.GetCgroupParamInt(cgroupCPUPath, cgroup.CFSQuotaFile)
	if err != nil {
		return 0, err
	}
	logs.Debugf("total cpu quota from cgroup: %d", totalCPUQuota)
	totalMilliCPUs := cgroup.QuotaToMilliCPU(totalCPUQuota, CPUPeriod)
	logs.Debugf("total milli cpu from cgroup: %d", totalMilliCPUs)
	if totalCPUQuota == cgroup.CPUNoLimits && milliCPUsConfiguration != 0 {
		totalCPUQuota = cpuQuotaConfiguration
		totalMilliCPUs = milliCPUsConfiguration
	}

	return totalMilliCPUs, nil
}

func (rc *ResourceControl) FrozenContainer(ID string) error {
	if !rc.cgroupManager.Exists(cgroup.CgroupName(ID)) {
		return runtimeErr.ErrCgroupNotExist{ID: ID}
	}
	state := api.Frozen
	c := &cgroup.CgroupConfig{
		Name: cgroup.CgroupName(ID),
		ResourceParameters: &runtimeapi.ResourceConfig{
			Freezer: &state,
		},
	}
	if err := rc.cgroupManager.Update(c); err != nil {
		return err
	}
	return nil
}

func (rc *ResourceControl) ThawContainer(ID string) error {
	if !rc.cgroupManager.Exists(cgroup.CgroupName(ID)) {
		return runtimeErr.ErrCgroupNotExist{ID: ID}
	}
	state := api.Thawed
	c := &cgroup.CgroupConfig{
		Name: cgroup.CgroupName(ID),
		ResourceParameters: &runtimeapi.ResourceConfig{
			Freezer: &state,
		},
	}
	if err := rc.cgroupManager.Update(c); err != nil {
		return err
	}
	return nil
}

func (rc *ResourceControl) ContainerResourceStats(ID string) (stats *api.ResourceStats, err error) {
	if !rc.cgroupManager.Exists(cgroup.CgroupName(ID)) {
		return nil, runtimeErr.ErrCgroupNotExist{ID: ID}
	}
	return rc.cgroupManager.GetResourceStats(cgroup.CgroupName(ID))
}

func (rc *ResourceControl) UpdateContainerResource(ID string, config *runtimeapi.ResourceConfig) error {
	cgName := cgroup.CgroupName(ID)
	if !rc.cgroupManager.Exists(cgName) {
		return runtimeErr.ErrCgroupNotExist{ID: ID}
	}
	cc := cgroup.CgroupConfig{
		Name:               cgName,
		ResourceParameters: config,
	}
	return rc.cgroupManager.Update(&cc)
}

func (rc *ResourceControl) ContainerResources(ID string) (resource *api.Resource, err error) {
	resource = &api.Resource{}
	cgName := cgroup.CgroupName(ID)
	if !rc.cgroupManager.Exists(cgName) {
		return nil, runtimeErr.ErrCgroupNotExist{ID: ID}
	}
	config, err := rc.cgroupManager.GetResourceConfig(cgName)
	if err != nil {
		return nil, err
	}
	resource.MilliCPUs = cgroup.QuotaToMilliCPU(*config.CpuQuota, int64(*config.CpuPeriod))
	resource.Memory = *config.Memory
	return resource, nil
}
