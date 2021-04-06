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

// Package api
package api

import "github.com/baidu/openless/pkg/api"

type ResourceParams struct {
	MemLimits   string
	CPURequests float64
	CPULimits   float64
}

// ResourceConfig holds information about all the supported cgroup resource parameters.
// port from k8s.io/kubernetes/pkg/kubelet/cm
type ResourceConfig struct {
	// Memory limit (in bytes).
	Memory *int64

	MemorySwap *int64
	// CPU shares (relative weight vs. other containers).
	CpuShares *uint64
	// CPU hardcap limit (in usecs). Allowed cpu time in a given period.
	CpuQuota *int64
	// CPU quota period.
	CpuPeriod *uint64
	// HugePageLimit map from page size (in bytes) to limit (in bytes)
	HugePageLimit map[int64]int64
	// FreezeState
	Freezer *api.FreezerState
}
