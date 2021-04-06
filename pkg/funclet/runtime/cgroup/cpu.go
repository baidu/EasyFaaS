// +build linux

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
	"strconv"
)

func GetCPUPeriod(path string) (CPUPeriod uint64, err error) {
	state, err := readFile(path, CFSPeriodFile)
	if err != nil {
		return 0, err
	}
	CPUPeriod, err = strconv.ParseUint(state, 10, 64)
	if err != nil {
		return 0, err
	}
	return CPUPeriod, nil
}

func GetCPUQuota(path string) (CPUQuota int64, err error) {
	state, err := readFile(path, CFSQuotaFile)
	if err != nil {
		return 0, err
	}
	CPUQuota, err = strconv.ParseInt(state, 10, 64)
	if err != nil {
		return 0, err
	}
	return CPUQuota, nil
}

func GetCPUShares(path string) (shares uint64, err error) {
	state, err := readFile(path, CPUShares)
	if err != nil {
		return 0, err
	}
	shares, err = strconv.ParseUint(state, 10, 64)
	if err != nil {
		return 0, err
	}
	return shares, nil
}
