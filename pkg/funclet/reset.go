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
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/baidu/easyfaas/pkg/api"
	funcletCtx "github.com/baidu/easyfaas/pkg/funclet/context"
	"github.com/baidu/easyfaas/pkg/funclet/runtime/runc"
)

func (f *Funclet) ResetContainerEvent(ctx *funcletCtx.Context, params *api.ResetRequest) (err error, response *api.ResetResponse) {
	containerID := params.ContainerID

	if _, err = f.ContainerManager.LockContainer(containerID, api.EventReset, ctx); err != nil {
		ctx.Logger.WithField("containerID", containerID).Errorf("get lock failed: %s", err)
		return err, nil
	}
	defer f.ContainerManager.UnLockContainerWithLog(containerID, ctx)

	if params.ScaleDownRecommendation != nil {
		for _, cID := range params.ScaleDownRecommendation.ResetContainers {
			if _, err = f.ContainerManager.LockContainer(cID, api.EventReset, ctx); err != nil {
				ctx.Logger.WithField("containerID", cID).Errorf("get lock failed: %s", err)
			}
			defer f.ContainerManager.UnLockContainerWithLog(cID, ctx)
		}

	}

	info, _ := f.ContainerManager.ContainerMap.GetContainer(containerID)
	ctx.SetContainer(info)

	return f.ResetContainers(ctx, params)
}

func (f *Funclet) ResetContainers(ctx *funcletCtx.Context, params *api.ResetRequest) (err error, response *api.ResetResponse) {
	ctx.Logger.Infof("start reset container")
	defer func() {
		if err != nil {
			ctx.Logger.Infof("reset container failed: %s", err)
		} else {
			ctx.Logger.Infof("reset container success")
		}
	}()

	var needScaleDown bool
	if params.ScaleDownRecommendation != nil {
		needScaleDown = true
		response = api.NewResetResponse()
	}

	// reset target container
	if err := f.ResetContainer(ctx); err != nil {
		if !needScaleDown {
			return err, nil
		}
		IDs := params.ScaleDownRecommendation.ResetContainers
		IDs = append(IDs, ctx.Container.ContainerID)

		if list, listErr := f.BulkGetContainers(IDs); listErr == nil {
			response.ScaleDownResult.Fails = list
		}
		return err, response
	}

	if !needScaleDown {
		return nil, nil
	}

	failedIDs := make([]string, 0)
	successIDs := make([]string, 0)
	wg := sync.WaitGroup{}
	wg.Add(len(params.ScaleDownRecommendation.ResetContainers))
	for _, cID := range params.ScaleDownRecommendation.ResetContainers {
		go func(ID string) {
			info, _ := f.ContainerManager.ContainerMap.GetContainer(ID)
			newCtx := funcletCtx.Context{RequestID: ctx.RequestID, Logger: ctx.Logger}
			newCtx.SetContainer(info)
			if err := f.ResetContainer(&newCtx); err != nil {
				failedIDs = append(failedIDs, ID)
			} else {
				successIDs = append(successIDs, ID)
			}
			wg.Done()
		}(cID)
	}
	wg.Wait()

	response.ScaleDownResult.Success = successIDs
	if list, listErr := f.BulkGetContainers(failedIDs); listErr == nil {
		response.ScaleDownResult.Fails = list
	}
	return nil, response
}

func (f *Funclet) ResetContainer(ctx *funcletCtx.Context) (err error) {
	ctx.Logger.Infof("start reset container")
	defer func() {
		if err != nil {
			ctx.Logger.Infof("reset container failed: %s", err)
		} else {
			ctx.Logger.Infof("reset container success")
		}
	}()
	containerID := ctx.Container.ContainerID

	var status runc.RuncContainerStatus
	containerStat, err := f.RuntimeClient.ContainerInfo(containerID)
	if err != nil || containerStat == nil {
		status = runc.ContainerStatusNotExist
	} else {
		ctx.Container.HostPid = containerStat.Pid
		status = containerStat.Status
	}

	if status == runc.ContainerStatusPausing || status == runc.ContainerStatusPaused {
		if err = f.RuntimeClient.ThawContainer(containerID); err != nil {
			ctx.Logger.Errorf("resume container %s failed: %+v", containerID, err)
			return fmt.Errorf("resume container %s failed: %+v", containerID, err)
		}
		status = runc.ContainerStatusRunning
	}

	if status == runc.ContainerStatusCreated || status == runc.ContainerStatusRunning {
		if err = f.StopContainer(ctx, true); err != nil {
			ctx.Logger.Errorf("kill container %s failed: %+v", containerID, err)
			return fmt.Errorf("kill container %s failed: %+v", containerID, err)
		}
		status = runc.ContainerStatusStopped
	}

	if status == runc.ContainerStatusStopped {
		if err := f.DeleteContainer(ctx); err != nil {

			ctx.Logger.Errorf("delete container %s failed %+v", containerID, err)
			return fmt.Errorf("delete container %s failed %+v", containerID, err)
		}
		status = runc.ContainerStatusNotExist
	}

	if status == runc.ContainerStatusNotExist {
		pid, err := f.GetPidFromFile(containerID)
		if err != nil {
			ctx.Logger.Warnf("container %s read from pid file failed: %s", containerID, err)
		}
		if pid != 0 {
			f.NetworkManager.UnsetContainerNet(pid)
		}

		cp, err := f.PathManager.GetPaths(containerID)
		if err != nil {
			ctx.Logger.Warnf("get container %s path failed: %s", containerID, err)
		}
		if cp != nil {
			if cp.PathName != "" {
				go f.RemoveTmpDevice(ctx, cp.PathName)
			}
			if err := f.PathManager.OutdatePaths(containerID); err != nil {
				ctx.Logger.Errorf("outdate container %s path failed: %s", containerID, err)
				return err
			}
		}
	}

	// init container
	if err := f.InitContainer(ctx); err != nil {
		return err
	}
	return nil
}

func (f *Funclet) DeleteContainer(ctx *funcletCtx.Context) (err error) {
	ctx.Logger.Infof("start delete container")
	defer func() {
		if err != nil {
			ctx.Logger.Infof("delete container failed: %s", err)
		} else {
			ctx.Logger.Infof("delete container success")
		}
	}()

	containerID := ctx.Container.ContainerID

	if err := f.RuntimeClient.RemoveContainer(containerID, false); err != nil {
		// TODO: unexpect error
		// Is this safe ?
		// delete force ?
		// check cgroup ?
		ctx.Logger.Errorf("delete container %s failed: %s; try to delete by force", containerID, err)
		if err := f.RuntimeClient.RemoveContainer(containerID, true); err != nil {
			ctx.Logger.Errorf("delete container %s failed: %s", containerID, err)
			return err
		}
	}
	return nil
}

func (f *Funclet) WaitForContainerExits(ctx *funcletCtx.Context) (err error) {
	ctx.Logger.Infof("start waiting for container to exits")
	defer func() {
		if err != nil {
			ctx.Logger.Infof("waiting for container to exits failed: %s", err)
		} else {
			ctx.Logger.Infof("waiting for container to exits success")
		}
	}()
	containerID := ctx.Container.ContainerID
	pid := ctx.Container.HostPid
	waitT := 1
	for {
		if waitT > f.Options.KillRuntimeWaitTime {
			return fmt.Errorf("waiting for container %s exit timeout", containerID)
		}
		// check process exists
		killErr := syscall.Kill(pid, syscall.Signal(0))
		// err expect to be "no such process"
		if killErr != nil && killErr != syscall.ESRCH {
			ctx.Logger.Errorf("container %s pid %d syscall unexpected err %s", containerID, pid, killErr)
		}
		procExists := killErr == nil || killErr == syscall.EPERM
		if !procExists {
			break
		}
		time.Sleep(time.Second)
		waitT++
	}
	return nil
}

func (f *Funclet) StopContainer(ctx *funcletCtx.Context, sync bool) (err error) {
	ctx.Logger.Infof("start stop container")
	defer func() {
		if err != nil {
			ctx.Logger.Infof("stop container failed: %s", err)
		} else {
			ctx.Logger.Infof("stop container success")
		}
	}()
	containerID := ctx.Container.ContainerID
	err = f.RuntimeClient.KillContainer(containerID, "SIGKILL", true)
	if err != nil {
		return fmt.Errorf("kill container %s failed: %+v", containerID, err)
	}

	if sync {
		if err = f.WaitForContainerExits(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (f *Funclet) GetPidFromFile(containerID string) (pid int, err error) {
	paths, err := f.PathManager.GetPaths(containerID)
	if err != nil {
		return 0, err
	}
	path := filepath.Join(paths.RunnerSpecPath, "runner.pid")
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err = strconv.Atoi(string(bytes.TrimSpace(d)))
	if err != nil {
		return 0, fmt.Errorf("error parsing pid from %s: %s", path, err)
	}
	return pid, nil
}

func (f *Funclet) RemoveTmpDevice(ctx *funcletCtx.Context, pathName string) {
	if err := f.TmpManager.RemoveTmpStorage(pathName); err != nil {
		ctx.Logger.Errorf("remove container tmp path %s failed: %s", pathName, err)
	}
}
