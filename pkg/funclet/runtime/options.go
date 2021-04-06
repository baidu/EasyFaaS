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

package runtime

import "github.com/spf13/pflag"

type ResourceOption struct {
	CgroupRootPath string
	TotalMemory    string
	TotalCPUs      float64
	ReservedMemory string
	ReservedCPUs   float64
	MemLimits      string
	CPURequests    float64
	CPULimits      float64
}

func NewResourceOption() *ResourceOption {
	return &ResourceOption{
		TotalCPUs:      10.0,
		ReservedMemory: "100M",
		ReservedCPUs:   0.1,
		MemLimits:      "128M",
		CPURequests:    0.2,
		CPULimits:      0.5,
	}
}

func (s *ResourceOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.CgroupRootPath, "cgroup-root-path", s.CgroupRootPath, "set cgroup root path; default /sys/fs/cgroup")
	fs.StringVar(&s.TotalMemory, "total-memory", s.TotalMemory, "set the total quota of memory limits; (units K,M,G) eg: 10K, 1M, 1G...")
	fs.Float64Var(&s.TotalCPUs, "total-cpu", s.TotalCPUs, "set the total quota of cpu limits; (units vCPU/Core) eg: 1 meanings one vCPU/core")
	fs.Float64Var(&s.ReservedCPUs, "reserved-cpu", s.ReservedCPUs, "set the reserved cpu resource; (units vCPU/Core) eg: 0.5 meanings half of a vCPU/core")
	fs.StringVar(&s.ReservedMemory, "reserved-memory", s.ReservedMemory, "set the reserved memory resource; (units K,M,G) eg: 10K, 1M, 1G...")
	fs.StringVar(&s.MemLimits, "runner-memory", s.MemLimits, "set the limit of memory for runtime; (units K,M,G) eg: 10K, 1M, 1G...")
	fs.Float64Var(&s.CPURequests, "runner-cpu-requests", s.CPURequests, "set a cpu request for runtime; (units vCPU/Core) eg: 0.2 meanings twenty percent of a vCPU/core")
	fs.Float64Var(&s.CPULimits, "runner-cpu-limits", s.CPULimits, "set a cpu limits for runtime; (units vCPU/Core) eg: 0.5 meanings half of a vCPU/core")
}
