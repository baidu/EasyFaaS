// +build !linux

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

// MilliCPUToQuota converts milliCPU to CFS quota and period values.
func MilliCPUToQuota(milliCPU int64, period int64) (quota int64) {
	return
}

// QuotaToMilliCPU
func QuotaToMilliCPU(quota int64, period int64) (milliCPU int64) {
	return
}

// MilliCPUToShares converts the milliCPU to CFS shares.
func MilliCPUToShares(milliCPU int64) int64 {
	return int64(0)
}

// GetCgroupSubsystems returns information about the mounted cgroup subsystems
func GetCgroupSubsystems() (*CgroupSubsystems, error) {
	return nil, nil
}

// getCgroupProcs takes a cgroup directory name as an argument
// reads through the cgroup's procs file and returns a list of tgid's.
// It returns an empty list if a procs file doesn't exists
func getCgroupProcs(dir string) ([]int, error) {
	return nil, nil
}

// parseUint
func parseUint(s string, base, bitSize int) (uint64, error) {
	return 0, nil
}

// parseInt
func parseInt(s string, base, bitSize int) (int64, error) {
	return 0, nil
}

// getCgroupParamKeyValue
func getCgroupParamKeyValue(t string) (string, uint64, error) {
	return "", 0, nil
}

// GetCgroupParamUint
func GetCgroupParamUint(cgroupPath, cgroupFile string) (uint64, error) {
	return 0, nil
}

// GetCgroupParamInt
func GetCgroupParamInt(cgroupPath, cgroupFile string) (int64, error) {
	return 0, nil
}

// getCgroupParamString
func getCgroupParamString(cgroupPath, cgroupFile string) (string, error) {
	return "", nil
}
