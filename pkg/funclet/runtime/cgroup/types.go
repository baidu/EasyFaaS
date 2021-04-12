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

package cgroup

import (
	"github.com/baidu/easyfaas/pkg/api"
	runtimeApi "github.com/baidu/easyfaas/pkg/funclet/runtime/api"
)

const (
	// Taken from lmctfy https://github.com/google/lmctfy/blob/master/lmctfy/controllers/cpu_controller.cc
	MinShares     = 2
	SharesPerCPU  = 1024
	MilliCPUToCPU = 1000

	// 100000 is equivalent to 100ms
	QuotaPeriod    = 100000
	MinQuotaPeriod = 1000

	// default memory limits is PAGE_COUNTER_MAX, now support 64 bit arch
	// refs: https://github.com/torvalds/linux/blob/master/include/linux/page_counter.h
	MemoryNoLimits = 0x7FFFFFFFFFFFF000
	CPUNoLimits    = -1
)

const (
	DefaultCgroupPath = "/sys/fs/cgroup"

	CFSPeriodFile    = "cpu.cfs_period_us"
	CFSQuotaFile     = "cpu.cfs_quota_us"
	CPUShares        = "cpu.shares"
	MemoryLimitsFile = "memory.limit_in_bytes"
)

// CgroupName is the abstract name of a cgroup prior to any driver specific conversion.
type CgroupName string

// CgroupConfig holds the cgroup configuration information.
// This is common object which is used to specify
// cgroup information to both systemd and raw cgroup fs
// implementation of the Cgroup Manager interface.
type CgroupConfig struct {
	// Fully qualified name prior to any driver specific conversions.
	Name CgroupName
	// ResourceParameters contains various cgroups settings to apply.
	ResourceParameters *runtimeApi.ResourceConfig
}

// CgroupManager allows for cgroup management.
// Supports Cgroup Creation ,Deletion and Updates.
type CgroupManager interface {
	// Create creates and applies the cgroup configurations on the cgroup.
	// It just creates the leaf cgroups.
	// It expects the parent cgroup to already exist.
	Create(*CgroupConfig) error
	// Destroy the cgroup.
	Destroy(*CgroupConfig) error
	// Update cgroup configuration.
	Update(*CgroupConfig) error
	// Exists checks if the cgroup already exists
	Exists(name CgroupName) bool
	// Name returns the literal cgroupfs name on the host after any driver specific conversions.
	// We would expect systemd implementation to make appropriate name conversion.
	// For example, if we pass /foo/bar
	// then systemd should convert the name to something like
	// foo.slice/foo-bar.slice
	Name(name CgroupName) string
	// CgroupName converts the literal cgroupfs name on the host to an internal identifier.
	CgroupName(name string) CgroupName
	// Pids scans through all subsytems to find pids associated with specified cgroup.
	Pids(name CgroupName) []int
	// ReduceCPULimits reduces the CPU CFS values to the minimum amount of shares.
	ReduceCPULimits(cgroupName CgroupName) error
	// GetResourceStats returns statistics of the specified cgroup as read from the cgroup fs.
	GetResourceStats(name CgroupName) (*api.ResourceStats, error)
	// GetResourcesConfig
	GetResourcesConfig(r *ResourceParams) (config *runtimeApi.ResourceConfig, err error)
	// GetCgroupSubsysPath
	GetCgroupSubsysPath(subsys string) (path string, err error)
	// GetResourceConfig
	GetResourceConfig(name CgroupName) (resourceConfig *runtimeApi.ResourceConfig, err error)
}

type ResourceParams struct {
	MemLimits   string
	CPURequests float64
	CPULimits   float64
}
