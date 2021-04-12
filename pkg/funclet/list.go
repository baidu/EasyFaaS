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
	runtimeapi "github.com/baidu/easyfaas/pkg/funclet/runtime/api"
	"github.com/baidu/easyfaas/pkg/funclet/runtime/runc"
)

func (f *Funclet) List(criteria *api.ListContainerCriteria) (list []*api.ContainerInfo, err error) {
	var getAll bool
	ids, err := criteria.ReadContainerIDs()
	if len(ids) == 0 {
		getAll = true
	}
	if getAll {
		return f.ListAllContainers()
	}
	cMap, err := f.BulkGetContainers(ids)
	if err != nil {
		return nil, err
	}
	for _, container := range cMap {
		list = append(list, container)
	}
	return list, nil
}

func (f *Funclet) ListAllContainers() (list []*api.ContainerInfo, err error) {
	// runc
	runtimeList, err := f.RuntimeClient.ListContainers()
	if err != nil {
		return nil, err
	}
	runtimeMap := make(map[string]*runtimeapi.Container)
	resourceMap := make(map[string]*api.Resource)
	for _, c := range runtimeList {
		runtimeMap[c.ID] = c
		// cgroup
		cr, err := f.RuntimeClient.ContainerResources(c.ID)
		if err != nil {
			return nil, err
		}
		resourceMap[c.ID] = cr
	}

	list = make([]*api.ContainerInfo, 0)
	f.ContainerManager.ContainerMap.CMap.Range(func(k, v interface{}) bool {
		if info, ok := runtimeMap[k.(string)]; ok {
			cacheInfo := v.(*api.ContainerInfo)
			resource := cacheInfo.Resource
			if rc, ok := resourceMap[info.ID]; ok {
				resource = rc
			}
			var isFrozen bool
			if info.Status == runc.ContainerStatusPaused {
				isFrozen = true
			}
			list = append(list, &api.ContainerInfo{
				Hostname:       info.ID,
				ContainerID:    info.ID,
				HostPid:        info.Pid,
				WithStreamMode: cacheInfo.WithStreamMode,
				IsFrozen:       isFrozen,
				Resource:       resource,
			})
		} else {
			cInfo := v.(*api.ContainerInfo)
			list = append(list, cInfo)
		}
		return true
	})
	return list, nil
}

func (f *Funclet) BulkGetContainers(IDs []string) (list map[string]*api.ContainerInfo, err error) {
	for _, ID := range IDs {
		if _, err := f.ContainerManager.ContainerMap.GetContainer(ID); err != nil {
			return nil, ContainerNotExist{ID: ID}
		}
	}
	list = make(map[string]*api.ContainerInfo, 0)
	for _, ID := range IDs {
		info, err := f.ContainerInfo(ID)
		if err != nil {
			return nil, err
		}
		list[info.ContainerID] = info
	}
	return
}
