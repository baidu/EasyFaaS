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

// Package funclet
package funclet

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/baidu/openless/pkg/util/id"

	"github.com/baidu/openless/pkg/util/logs"

	"github.com/baidu/openless/pkg/funclet/tmp"
)

func (f *Funclet) RecycleTask(stopCh <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			f.logger.Infof("funclet stopped")
			// graceful exit task
			// ...
			list, _ := f.RuntimeClient.ListContainers()
			for _, container := range list {
				ID := container.ID
				f.RuntimeClient.ThawContainer(ID)
			}
			return
		case <-ticker.C:
			taskID := id.GetTaskID()
			taskLogger := f.logger.WithField("task-id", taskID)
			taskLogger.Infof("start recycle task")
			defer taskLogger.Infof("finish recycle task")
			if err := f.tmpRecycleTask(taskLogger); err != nil {
				taskLogger.Errorf("tmp recycle task failed: %s", err)
			}
			// other recycle task
			// ...
		}
	}
}

func (f *Funclet) tmpRecycleTask(tlog *logs.Logger) (err error) {
	tlog.Infof("[tmp recycle task] start task")
	defer tlog.Infof("[tmp recycle task] finish task")
	defer func() {
		if err := recover(); err != nil {
			tlog.Errorf("[tmp recycle task] occurred panic: %s, stask [%s]", err, debug.Stack())
		}
	}()
	ids := make([]string, 0)
	f.ContainerManager.ContainerMap.CMap.Range(func(k, v interface{}) bool {
		cid, ok := k.(string)
		if !ok {
			tlog.Errorf("containers map key is invalid, key %+v", k)
			err = fmt.Errorf("[tmp recycle task] failed: containers map key is invalid")
			return false
		}
		ids = append(ids, cid)
		return true
	})
	if err != nil {
		return
	}
	tlog.V(9).Infof("[tmp recycle task] all container ids %v", ids)

	// TODO: Try more aggressive way!
	paths, err := f.TmpManager.SnapshotTmpPaths()
	if err != nil {
		tlog.Errorf("[tmp recycle task] shapshot tmp path failed: %s", err)
		return
	}
	tlog.V(9).Infof("[tmp recycle task] all tmp paths %v", paths)
	recycleList := make([]string, 0)
	for _, path := range *paths {
		pathName := strings.TrimPrefix(path, tmp.RunnersTmpPath)
		pathName = strings.TrimLeft(pathName, "/")
		cID, err := f.PathManager.GetContainerID(pathName)
		if err != nil {
			tlog.Errorf("[tmp recycle task] get container name by path name failed: %s", err)
			continue
		}
		cp, err := f.PathManager.GetPaths(cID)
		if err != nil {
			tlog.Errorf("[tmp recycle task] get container paths by container id failed: %s", err)
			continue
		}
		if cp.PathName == "" {
			tlog.Error("[tmp recycle task] get container paths by container id failed: path name is empty")
			continue
		}
		if pathName != cp.PathName {
			recycleList = append(recycleList, pathName)
		}
	}
	tlog.V(4).Infof("[tmp recycle task] all recycle paths %v", recycleList)
	for _, pathName := range recycleList {
		tlog.V(6).Infof("[tmp recycle task] remove tmp storage path %s ", pathName)
		if err := f.TmpManager.RemoveTmpStorage(pathName); err != nil {
			tlog.Errorf("[tmp recycle task] remove tmp storage %s failed: %s", pathName, err)
			continue
		}
	}
	return nil
}
