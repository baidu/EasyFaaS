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
	"errors"
	"sync"

	funcletCtx "github.com/baidu/easyfaas/pkg/funclet/context"

	"github.com/baidu/easyfaas/cmd/funclet/options"

	"github.com/baidu/easyfaas/pkg/api"
)

var (
	ContainerNotExists = errors.New("container not exists")
)

type ContainerManager struct {
	ContainerMap *ContainerMap
}

func NewContainerManager(podName string, o *options.FuncletOptions) *ContainerManager {
	return &ContainerManager{
		ContainerMap: InitContainerMap(podName, o.ContainerNum),
	}
}

func (m *ContainerManager) LockContainer(id string, event api.Event, ctx *funcletCtx.Context) (res bool, err error) {
	info, err := m.ContainerMap.GetContainer(id)
	if err != nil {
		ctx.Logger.Errorf("get container %s status err %s", id, err)
		return
	}
	res = info.EventLock.Lock()
	if res {
		ctx.Logger.V(9).Infof("get lock success, container %s event %s", id, event)
		info.CurrentEvent = event
	} else {
		err = &ContainerIsBusy{
			ID:           id,
			CurrentEvent: info.CurrentEvent,
			TriggerEvent: event,
		}
	}
	return
}

func (m *ContainerManager) UnLockContainer(id string, ctx *funcletCtx.Context) (err error) {
	info, err := m.ContainerMap.GetContainer(id)
	if err != nil {
		ctx.Logger.Errorf("get container %s status err %s", id, err)
		return
	}
	info.EventLock.UnLock()
	return
}

func (m *ContainerManager) UnLockContainerWithLog(id string, ctx *funcletCtx.Context) (err error) {
	ctx.Logger.V(9).Infof("start unlock container %s", id)
	defer ctx.Logger.V(9).Infof("finish unlock container %s", id)
	info, err := m.ContainerMap.GetContainer(id)
	if err != nil {
		ctx.Logger.Errorf("get container %s status err %s", id, err)
		return
	}
	info.EventLock.UnLock()
	return
}

type ContainerMap struct {
	CMap sync.Map
}

func (m *ContainerMap) GetContainer(id string) (info *api.ContainerInfo, err error) {
	if info, exist := m.Exist(id); exist {
		return info, nil
	}
	return nil, ContainerNotExists
}

func (m *ContainerMap) Exist(id string) (info *api.ContainerInfo, exist bool) {
	val, ok := m.CMap.Load(id)
	if !ok {
		return nil, false
	}
	info, ok = val.(*api.ContainerInfo)
	if !ok {
		return nil, false
	}
	return info, true
}

func (m *ContainerMap) UpdateContainerPid(id string, pid int) (err error) {
	info, exist := m.Exist(id)
	if !exist {
		return ContainerNotExists
	}
	info.HostPid = pid
	return nil
}

func (m *ContainerMap) UpdateContainerStreamMode(id string, streamMode bool) (err error) {
	info, exist := m.Exist(id)
	if !exist {
		return ContainerNotExists
	}
	info.WithStreamMode = streamMode
	return nil
}

func InitContainerMap(podName string, num int) *ContainerMap {
	cMap := &ContainerMap{
		CMap: sync.Map{},
	}
	for i := 0; i < num; i++ {
		cID := generateContainerID(podName, i)
		cMap.CMap.Store(cID, &api.ContainerInfo{
			Hostname:    cID,
			ContainerID: cID,
			EventLock:   api.NewEventLock(),
		})
	}
	return cMap
}
