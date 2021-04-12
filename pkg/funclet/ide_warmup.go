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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/baidu/easyfaas/pkg/api"
	funcletCtx "github.com/baidu/easyfaas/pkg/funclet/context"
	"github.com/baidu/easyfaas/pkg/funclet/file"
	"github.com/baidu/easyfaas/pkg/funclet/runtime/runc"
	"github.com/baidu/easyfaas/pkg/util/strtool"
)

func (f *Funclet) IDEWarmUpContainerEvent(ctx *funcletCtx.Context, params api.WarmupRequest) (err error) {
	containerID := params.ContainerID
	if _, err := f.ContainerManager.LockContainer(containerID, api.EventWarmup, ctx); err != nil {
		ctx.Logger.WithField("containerID", containerID).Errorf("get lock failed: %s", err)
		return err
	}

	defer f.ContainerManager.UnLockContainerWithLog(containerID, ctx)
	info, _ := f.ContainerManager.ContainerMap.GetContainer(params.ContainerID)
	ctx.SetContainer(info)

	return f.IDEWarmUp(ctx, params)
}

func (f *Funclet) IDEWarmUp(ctx *funcletCtx.Context, params api.WarmupRequest) (err error) {
	ctx.Logger.Infof("start warmup container")
	defer func() {
		if err != nil {
			ctx.Logger.Infof("warmup container failed: %s", err)
		} else {
			ctx.Logger.Infof("warmup container success")
		}
	}()
	containerID := params.ContainerID
	containerStat, err := f.RuntimeClient.ContainerInfo(containerID)
	if err != nil {
		return ContainerNotExist{ID: containerID}
	}

	if containerStat.Status != runc.ContainerStatusRunning {
		return ContainerNotRunning{ID: containerID}
	}

	// 1. pid
	pid := containerStat.Pid
	if ctx.Container.HostPid != pid {
		f.ContainerManager.ContainerMap.UpdateContainerPid(containerID, pid)
	}

	// 2. function-meta
	codeSha256 := params.Configuration.CodeSha256
	hexCodeSha256, _ := strtool.Base64ToHex(codeSha256)
	if params.Code == nil || params.WarmUpContainerArgs.Code.Location == "" {
		return errors.New("code location was empty")
	}
	mountInfo := &MountInfo{
		RuntimePath: params.WarmUpContainerArgs.RuntimeConfiguration.Path,
	}
	// 3.1 fetch code
	mountInfo.CodePath = f.GetUserCodePath(hexCodeSha256)
	if err := os.MkdirAll(mountInfo.CodePath, os.ModePerm); err != nil {
		return err
	}
	codeChain := make(chan error, 1)
	go func() {
		defer close(codeChain)
		var err error
		_, err = f.prepareUserCode(ctx, params.WarmUpContainerArgs.Code,
			codeSha256, hexCodeSha256)
		if err != nil {
			codeChain <- err
			return
		}
		codeChain <- nil
	}()

	// mount source-target ro file-type
	mountPairs := make([]*file.MountPair, 0)
	containerPaths, err := f.PathManager.GetPaths(containerID)
	if err != nil {
		return err
	}

	mountPairs = append(mountPairs, &file.MountPair{
		Source: containerPaths.CodeWorkspacePath,
		Target: containerPaths.DataCodePath,
	})

	mountPairs = append(mountPairs, &file.MountPair{
		Source: mountInfo.RuntimePath,
		Target: containerPaths.DataRuntimePath,
	})

	ctx.Logger.V(9).Infof("mount pairs %+v", mountPairs)

	if err := f.MountManager.BindMount(mountPairs, true); err != nil {
		ctx.Logger.Errorf("container %s mount failed : %s", containerID, err)
		return err
	}

	t := time.NewTicker(10 * time.Second)
Loop:
	for {
		select {
		case <-t.C:
			ctx.Logger.Errorf("download code timeout 10 seconds")
			return errors.New("download code timeout 10 seconds")
		case err, ok := <-codeChain:
			if err != nil {
				ctx.Logger.Errorf("download code err: %+v", err)
				return fmt.Errorf("download code err: %+v", err)
			}
			if !ok {
				break Loop
			}
		}
	}

	if err := os.Chmod(containerPaths.CodeWorkspacePath, os.ModePerm); err != nil {
		ctx.Logger.Errorf("code workspace chmod failed: %s", err)
		return err
	}
	bsamCodePath := filepath.Join(containerPaths.CodeWorkspacePath, params.Configuration.FunctionName)
	ctx.Logger.V(8).Infof("copy code cache %s to workspace %s", mountInfo.CodePath, bsamCodePath)

	if err := CopyDir(mountInfo.CodePath, bsamCodePath, os.ModePerm); err != nil {
		ctx.Logger.Errorf("copy code failed: %s", err)
		return err
	}

	// generate bsam yaml
	bsamPath := filepath.Join(containerPaths.CodeWorkspacePath, "template.yaml")
	ctx.Logger.V(9).Infof("bsam file path %s", bsamPath)
	template := f.generateBsamTemplate(ctx, params)
	data, err := yaml.Marshal(template)
	if err != nil {
		ctx.Logger.Errorf("generate bsam data failed: %s", err)
		return err
	}
	if err := ioutil.WriteFile(bsamPath, data, 0777); err != nil {
		ctx.Logger.Errorf("create bsam file failed: %s", err)
		return err
	}
	return nil
}

func (f *Funclet) generateBsamTemplate(ctx *funcletCtx.Context, params api.WarmupRequest) *BsamTemplate {
	template := NewBsamTemplate()
	template.Resources[params.Configuration.FunctionName] = NewBsamFunctionResource(params.Configuration)
	ctx.Logger.Debugf("bsam template %v", template)
	return template
}
