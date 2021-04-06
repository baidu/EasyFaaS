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
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	libcontainercgroups "github.com/opencontainers/runc/libcontainer/cgroups"
)

var (
	ErrNotValidFormat = errors.New("line is not a valid key value format")
)

// MilliCPUToQuota converts milliCPU to CFS quota and period values.
func MilliCPUToQuota(milliCPU int64, period int64) (quota int64) {
	// CFS quota is measured in two values:
	//  - cfs_period_us=100ms (the amount of time to measure usage across)
	//  - cfs_quota=20ms (the amount of cpu time allowed to be used across a period)
	// so in the above example, you are limited to 20% of a single CPU
	// for multi-cpu environments, you just scale equivalent amounts

	if milliCPU == 0 {
		return
	}

	// we set the period to 100ms by default
	if period == 0 {
		period = QuotaPeriod
	}

	// we then convert your milliCPU to a value normalized over a period
	quota = (milliCPU * period) / MilliCPUToCPU

	// quota needs to be a minimum of 1ms.
	if quota < MinQuotaPeriod {
		quota = MinQuotaPeriod
	}

	return
}

// QuotaToMilliCPU converts CFS quota and period values to milliCPU.
func QuotaToMilliCPU(quota int64, period int64) (milliCPU int64) {
	// no limit
	if quota < 0 {
		milliCPU = -1
		return
	}
	// we set the period to 100ms by default
	if period == 0 {
		period = QuotaPeriod
	}

	milliCPU = (quota * MilliCPUToCPU) / period
	return
}

// MilliCPUToShares converts the milliCPU to CFS shares.
func MilliCPUToShares(milliCPU int64) uint64 {
	if milliCPU == 0 {
		// Docker converts zero milliCPU to unset, which maps to kernel default
		// for unset: 1024. Return 2 here to really match kernel default for
		// zero milliCPU.
		return MinShares
	}
	// Conceptually (milliCPU / milliCPUToCPU) * sharesPerCPU, but factored to improve rounding.
	shares := (milliCPU * SharesPerCPU) / MilliCPUToCPU
	if shares < MinShares {
		return MinShares
	}
	return uint64(shares)
}

// GetCgroupSubsystems returns information about the mounted cgroup subsystems
func GetCgroupSubsystems() (*CgroupSubsystems, error) {
	// get all cgroup mounts.
	allCgroups, err := libcontainercgroups.GetCgroupMounts(true)
	if err != nil {
		return &CgroupSubsystems{}, err
	}
	if len(allCgroups) == 0 {
		return &CgroupSubsystems{}, fmt.Errorf("failed to find cgroup mounts")
	}
	mountPoints := make(map[string]string, len(allCgroups))
	for _, mount := range allCgroups {
		for _, subsystem := range mount.Subsystems {
			mountPoints[subsystem] = mount.Mountpoint
		}
	}
	return &CgroupSubsystems{
		Mounts:      allCgroups,
		MountPoints: mountPoints,
	}, nil
}

// getCgroupProcs takes a cgroup directory name as an argument
// reads through the cgroup's procs file and returns a list of tgid's.
// It returns an empty list if a procs file doesn't exists
func getCgroupProcs(dir string) ([]int, error) {
	procsFile := filepath.Join(dir, "cgroup.procs")
	f, err := os.Open(procsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// The procsFile does not exist, So no pids attached to this directory
			return []int{}, nil
		}
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	out := []int{}
	for s.Scan() {
		if t := s.Text(); t != "" {
			pid, err := strconv.Atoi(t)
			if err != nil {
				return nil, fmt.Errorf("unexpected line in %v; could not convert to pid: %v", procsFile, err)
			}
			out = append(out, pid)
		}
	}
	return out, nil
}

// Saturates negative values at zero and returns a uint64.
// Due to kernel bugs, some of the memory cgroup stats can be negative.
func parseUint(s string, base, bitSize int) (uint64, error) {
	value, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		intValue, intErr := strconv.ParseInt(s, base, bitSize)
		// 1. Handle negative values greater than MinInt64 (and)
		// 2. Handle negative values lesser than MinInt64
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if intErr != nil && intErr.(*strconv.NumError).Err == strconv.ErrRange && intValue < 0 {
			return 0, nil
		}

		return value, err
	}

	return value, nil
}

func parseInt(s string, base, bitSize int) (int64, error) {
	intValue, intErr := strconv.ParseInt(s, base, bitSize)
	if intErr != nil {
		return 0, intErr
	}
	return intValue, nil
}

// Parses a cgroup param and returns as name, value
//  i.e. "io_service_bytes 1234" will return as io_service_bytes, 1234
func getCgroupParamKeyValue(t string) (string, uint64, error) {
	parts := strings.Fields(t)
	switch len(parts) {
	case 2:
		value, err := parseUint(parts[1], 10, 64)
		if err != nil {
			return "", 0, fmt.Errorf("unable to convert param value (%q) to uint64: %v", parts[1], err)
		}

		return parts[0], value, nil
	default:
		return "", 0, ErrNotValidFormat
	}
}

// Gets a single uint64 value from the specified cgroup file.
func GetCgroupParamUint(cgroupPath, cgroupFile string) (uint64, error) {
	fileName := filepath.Join(cgroupPath, cgroupFile)
	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return 0, err
	}
	trimmed := strings.TrimSpace(string(contents))
	if trimmed == "max" {
		return math.MaxUint64, nil
	}

	res, err := parseUint(trimmed, 10, 64)
	if err != nil {
		return res, fmt.Errorf("unable to parse %q as a uint from Cgroup file %q", string(contents), fileName)
	}
	return res, nil
}

// Gets a single int64 value from the specified cgroup file.
func GetCgroupParamInt(cgroupPath, cgroupFile string) (int64, error) {
	fileName := filepath.Join(cgroupPath, cgroupFile)
	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return 0, err
	}
	trimmed := strings.TrimSpace(string(contents))
	if trimmed == "max" {
		return math.MaxInt64, nil
	}

	res, err := parseInt(trimmed, 10, 64)
	if err != nil {
		return res, fmt.Errorf("unable to parse %q as a uint from Cgroup file %q", string(contents), fileName)
	}
	return res, nil
}

// Gets a string value from the specified cgroup file
func getCgroupParamString(cgroupPath, cgroupFile string) (string, error) {
	contents, err := ioutil.ReadFile(filepath.Join(cgroupPath, cgroupFile))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}
