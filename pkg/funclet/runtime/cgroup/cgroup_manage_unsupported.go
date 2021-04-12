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

import "github.com/baidu/easyfaas/pkg/api"

// CgroupSubsystems holds information about the mounted cgroup subsytems
type CgroupSubsystems struct {
	MountPoints map[string]string
}

// cgroupManagerImpl implements the CgroupManager interface.
// Its a stateless object which can be used to
// update,create or delete any number of cgroups
// It uses the Libcontainer raw fs cgroup manager for cgroup management.
type cgroupManagerImpl struct {
}

// NewCgroupManager is a factory method that returns a CgroupManager
func NewCgroupManager(cs *CgroupSubsystems, cgroupPath, cgroupDriver string) (m CgroupManager, err error) {
	return nil, nil
}

// Destroy destroys the specified cgroup
func (m *cgroupManagerImpl) Freeze(cgroupConfig *CgroupConfig, state *api.FreezerState) error {
	return nil
}
