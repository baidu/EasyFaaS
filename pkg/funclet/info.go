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

package funclet

import (
	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/funclet/runtime/runc"
)

func (f *Funclet) ContainerInfo(ID string) (info *api.ContainerInfo, err error) {
	// cache
	cacheInfo, exist := f.ContainerManager.ContainerMap.Exist(ID)
	if !exist {
		return nil, ContainerNotExist{ID: ID}
	}
	// runc
	runtimeInfo, runcErr := f.RuntimeClient.ContainerInfo(ID)

	// cgroups
	cr, resourceErr := f.RuntimeClient.ContainerResources(ID)
	if resourceErr != nil {
		return nil, resourceErr
	}

	// runc state failed, return container info from cache
	if runcErr != nil {
		cacheInfo.Resource = cr
		return cacheInfo, nil
	}

	isFrozen := cacheInfo.IsFrozen
	if runtimeInfo.Status == runc.ContainerStatusPaused {
		isFrozen = true
	}
	stats, err := f.RuntimeClient.ContainerResourceStats(ID)
	if err != nil {
		return nil, err
	}

	return &api.ContainerInfo{
		Hostname:       runtimeInfo.ID,
		ContainerID:    runtimeInfo.ID,
		HostPid:        runtimeInfo.Pid,
		WithStreamMode: cacheInfo.WithStreamMode,
		IsFrozen:       isFrozen,
		Resource:       cr,
		ResourceStats:  stats,
	}, nil
}
