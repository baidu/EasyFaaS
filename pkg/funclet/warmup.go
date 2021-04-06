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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	runtimeapi "github.com/baidu/openless/pkg/funclet/runtime/api"

	"go.uber.org/zap"
	"github.com/baidu/openless/pkg/api"
	funcletCtx "github.com/baidu/openless/pkg/funclet/context"
	"github.com/baidu/openless/pkg/funclet/file"
	"github.com/baidu/openless/pkg/funclet/runtime/runc"
	"github.com/baidu/openless/pkg/util/json"
	"github.com/baidu/openless/pkg/util/strtool"
)

const RuntimeHTTPSock = ".runtime-http.sock"

var bufferMemorySize = int64(1 * 1024 * 1024)

func (f *Funclet) WarmUpContainerEvent(ctx *funcletCtx.Context, params api.WarmupRequest) (err error) {
	containerID := params.ContainerID
	if _, err := f.ContainerManager.LockContainer(containerID, api.EventWarmup, ctx); err != nil {
		ctx.Logger.WithField("containerID", containerID).Errorf("get lock failed: %s", err)
		return err
	}

	defer f.ContainerManager.UnLockContainerWithLog(containerID, ctx)
	info, _ := f.ContainerManager.ContainerMap.GetContainer(params.ContainerID)
	ctx.SetContainer(info)

	return f.WarmUp(ctx, params)
}

func (f *Funclet) WarmUp(ctx *funcletCtx.Context, params api.WarmupRequest) (err error) {
	ctx.Logger.Infof("start warmup container")
	defer func() {
		if err != nil {
			ctx.Logger.Infof("warmup container failed: %s", err)
		} else {
			ctx.Logger.Infof("warmup container success")
		}
	}()
	containerID := params.ContainerID
	if params.ScaleUpRecommendation != nil {
		if err := f.scaleUp(ctx, params.ScaleUpRecommendation); err != nil {
			return err
		}
	}
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
	// 3.2 prepare for the runtime configuration
	mountInfo.ConfPath, err = f.prepareRuntimeConf(ctx, params.WarmUpContainerArgs)
	if err != nil {
		return err
	}

	if params.WithStreamMode || strings.HasSuffix(params.WarmUpContainerArgs.RuntimeConfiguration.Name, "stream") {
		f.ContainerManager.ContainerMap.UpdateContainerStreamMode(containerID, true)
	}

	// mount source-target ro file-type
	mountPairs := make([]*file.MountPair, 0)
	containerPaths, err := f.PathManager.GetPaths(containerID)
	if err != nil {
		return err
	}
	mountPairs = append(mountPairs, &file.MountPair{
		Source: mountInfo.CodePath,
		Target: containerPaths.DataCodePath,
	})

	mountPairs = append(mountPairs, &file.MountPair{
		Source: mountInfo.ConfPath,
		Target: containerPaths.DataConfigPath,
	})

	mountPairs = append(mountPairs, &file.MountPair{
		Source: mountInfo.RuntimePath,
		Target: containerPaths.DataRuntimePath,
	})

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
				// send signal
				if err := syscall.Kill(pid, syscall.SIGUSR1); err != nil {
					ctx.Logger.Errorf("set init signal err: %+v", err)
					return err
				}
				break Loop
			}
		}
	}

	return nil
}

// GetUserCodePath
func (f *Funclet) GetUserCodePath(codeSha256 string) string {
	return filepath.Join(f.Options.CachePath, codeSha256)
}

func (f *Funclet) prepareUserCode(ctx *funcletCtx.Context, code *api.CodeStorage, codeSha256, hexCodeSha256 string) (string, error) {
	destination := f.GetUserCodePath(hexCodeSha256)
	codeFolder := f.Options.CachePath
	defer ctx.Logger.TimeTrack(time.Now(), "Prepare user code",
		zap.String("codesha256", codeSha256),
		zap.String("hexcode", hexCodeSha256),
		zap.String("repositoryType", "bos"),
	)

	// serial fetch code
	prepareCodeChain, isMaster := f.HandlingChanMap.GetChan(hexCodeSha256)
	if !isMaster {
		<-prepareCodeChain
	} else {
		defer f.HandlingChanMap.CloseChan(hexCodeSha256)
	}

	// check code
	foundCode := f.CodeManager.FindCode(codeFolder, hexCodeSha256)
	foundCodeCompleteTag := f.CodeManager.FindCodeCompeleteTag(codeFolder, hexCodeSha256)
	if foundCode && foundCodeCompleteTag {
		ctx.Logger.V(6).Infof("Code cache %s found", destination)
		return destination, nil
	}

	zipFilePath, err := f.CodeManager.FetchCode(ctx, code)
	if err != nil {
		return "", err
	}
	defer os.Remove(zipFilePath)

	// check CodeSha256
	if !f.CodeManager.CheckCode(ctx, zipFilePath, codeSha256) {
		ctx.Logger.Warnf("Code %s checksum failed", zipFilePath)
		return "", errors.New("CodeSha256 checksum failed")
	}

	// unzip code
	if err := f.CodeManager.UnzipCode(ctx, zipFilePath, destination); err != nil {
		return "", err
	}

	// mark as finished
	if err := f.CodeManager.CreateCodeCompeleteTag(codeFolder, hexCodeSha256); err != nil {
		return "", err
	}
	return destination, nil
}

func (f *Funclet) prepareRuntimeConf(ctx *funcletCtx.Context, warmUpContainerArgs *api.WarmUpContainerArgs) (string, error) {
	confPath := f.GetContainerConfPath(ctx.Container.Hostname)
	if err := os.MkdirAll(confPath, os.ModePerm); err != nil {
		return "", err
	}
	runtimeConfPath := filepath.Join(confPath, "runtime.conf")
	envConfPath := filepath.Join(confPath, "env.conf")
	metaConfPath := filepath.Join(confPath, "meta.conf")
	confFile, err := os.Create(runtimeConfPath)
	if err != nil {
		return "", err
	}
	defer confFile.Close()

	confFile.WriteString(fmt.Sprintf("%s\n", warmUpContainerArgs.RuntimeConfiguration.Bin))
	for _, arg := range warmUpContainerArgs.RuntimeConfiguration.Args {
		confFile.WriteString(fmt.Sprintf("%s\n", arg))
	}

	var variables map[string]string
	if warmUpContainerArgs.Configuration.Environment == nil || warmUpContainerArgs.Configuration.Environment.Variables == nil {
		variables = make(map[string]string)
	} else {
		variables = warmUpContainerArgs.Configuration.Environment.Variables
	}
	if warmUpContainerArgs.WithStreamMode || strings.HasSuffix(warmUpContainerArgs.RuntimeConfiguration.Name, "stream") {
		variables["BCE_CFC_RUNTIME_MODE"] = "stream"
		variables["BCE_CFC_HTTP_SOCKET"] = filepath.Join(f.Options.RunnerSpecOption.TargetRuntimeSocketPath, RuntimeHTTPSock)
	}
	f.prepareEnvConf(envConfPath, ctx.Container.Hostname, variables)

	mf, err := os.Create(metaConfPath)
	if err != nil {
		return "", err
	}
	meta := &Meta{
		FunctionConfig: warmUpContainerArgs.Configuration,
		RuntimePath:    warmUpContainerArgs.RuntimeConfiguration.Path,
	}

	j, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	mf.Write(j)
	return confPath, nil
}

func (f *Funclet) prepareEnvConf(envConfPath, podName string, environment map[string]string) error {
	ef, err := os.Create(envConfPath)
	if err != nil {
		return err
	}
	defer ef.Close()
	environment["BCE_RUNTIME_START_TIME"] = strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	environment["BCE_USER_CODE_ROOT"] = f.Options.RunnerSpecOption.TargetCodePath
	environment["BCE_RUNTIME_INVOKER_SOCKS"] = f.Options.InvokerSocks
	environment["BCE_RUNTIME_FUNCLET_SOCKS"] = f.Options.ListenPath
	environment["BCE_RUNTIME_NAME"] = podName
	if f.Options.ListenType == "tcp" {
		environment["BCE_RUNTIME_INVOKER_PORT"] = fmt.Sprint(f.Options.InvokerDispatcherPort)
	}

	e, err := json.Marshal(environment)
	if err != nil {
		return err
	}
	ef.Write(e)
	return nil
}

func (f *Funclet) GetContainerConfPath(containerName string) string {
	return filepath.Join(f.Options.ConfPath, containerName)
}

func (f *Funclet) scaleUp(ctx *funcletCtx.Context, recommend *api.ScaleUpRecommendation) error {
	ctx.Logger.Infof("start scale up: recommend [%v]", recommend)
	defer ctx.Logger.Infof("finish scale up: recommend [%v]", recommend)
	ids := append(recommend.MergedContainers, recommend.TargetContainer)
	for _, id := range ids {
		if _, err := f.RuntimeClient.ContainerInfo(id); err != nil {
			return ContainerNotExist{ID: id}
		}
	}

	for _, id := range recommend.MergedContainers {
		if err := f.RuntimeClient.FrozenContainer(id); err != nil {
			return err
		}
		if err := f.scaleDownContainerToMinimum(id); err != nil {
			return err
		}
	}
	config, err := f.RuntimeClient.GetResourceConfigByReadableMemory(recommend.TargetMemory)
	if err != nil {
		return err
	}
	return f.scaleUpContainer(ctx.Container.ContainerID, config)
}

func (f *Funclet) scaleDownContainerToMinimum(ID string) error {
	resourceStats, err := f.RuntimeClient.ContainerResourceStats(ID)
	if err != nil {
		return err
	}
	limit := f.getMinMemoryLimit(resourceStats.MemoryStats.Usage)
	rc := &runtimeapi.ResourceConfig{
		Memory: &limit,
	}
	return f.RuntimeClient.UpdateContainerResource(ID, rc)
}

func (f *Funclet) scaleDownContainerToDefault(ID string) error {
	resourceStats, err := f.RuntimeClient.ContainerResourceStats(ID)
	if err != nil {
		return err
	}
	rc := f.RuntimeClient.GetDefaultResourceConfig()
	if resourceStats.MemoryStats.Usage > *rc.Memory {
		return fmt.Errorf("container memory usage is overused: current usage %d, default memory limit %d", resourceStats.MemoryStats.Usage, *rc.Memory)
	}
	return f.RuntimeClient.UpdateContainerResource(ID, rc)
}

func (f *Funclet) scaleUpContainer(ID string, config *runtimeapi.ResourceConfig) error {
	resourceStats, err := f.RuntimeClient.ContainerResourceStats(ID)
	if err != nil {
		return err
	}
	if config.Memory != nil && *config.Memory < resourceStats.MemoryStats.Limit {
		return fmt.Errorf("set memory resource invaild")
	}
	return f.RuntimeClient.UpdateContainerResource(ID, config)
}

func (f *Funclet) getMinMemoryLimit(usage int64) (limit int64) {
	return usage + bufferMemorySize
}
