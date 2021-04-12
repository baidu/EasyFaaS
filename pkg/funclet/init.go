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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/baidu/easyfaas/pkg/api"
	funcletCtx "github.com/baidu/easyfaas/pkg/funclet/context"
	"github.com/baidu/easyfaas/pkg/funclet/file"
	"github.com/baidu/easyfaas/pkg/funclet/runner"
	runtimeapi "github.com/baidu/easyfaas/pkg/funclet/runtime/api"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

func (f *Funclet) InitContainerEvent(containerID string) (err error) {
	ctx := &funcletCtx.Context{
		Logger: f.logger,
	}
	if _, err := f.ContainerManager.LockContainer(containerID, api.EventInit, ctx); err != nil {
		f.logger.WithField("containerID", containerID).Errorf("get lock failed: %s", err)
		return err
	}
	defer f.ContainerManager.UnLockContainerWithLog(containerID, ctx)

	info, _ := f.ContainerManager.ContainerMap.GetContainer(containerID)
	ctx.SetContainer(info)

	return f.InitContainer(ctx)
}

func (f *Funclet) InitContainer(ctx *funcletCtx.Context) (err error) {
	ctx.Logger.Infof("start init container")
	defer func() {
		if err != nil {
			ctx.Logger.Infof("init container failed: %s", err)
		} else {
			ctx.Logger.Infof("init container success")
		}
	}()
	containerID := ctx.Container.ContainerID

	// prepare runner config & data directory
	cp, err := f.PathManager.GeneratePaths(containerID)
	if err != nil {
		ctx.Logger.Errorf("container %s prepare dir failed: %s", containerID, err)
		return err
	}

	// get tmp device
	if _, err := f.TmpManager.GetTmpStorage(cp.PathName); err != nil {
		ctx.Logger.Errorf("prepare container %s tmp dir failed: %v", containerID, err)
		return err
	}

	// prepare runner config file
	if err = runner.PrepareConfigDir(containerID, cp.SpecConfigPath); err != nil {
		ctx.Logger.Errorf("prepare config dir failed: %v", err)
		return err
	}

	// prepare runc spec config.json & network file (eg: /etc/hosts)
	if err = f.prepareConfigData(containerID, cp); err != nil {
		ctx.Logger.Errorf("prepare container %s config data failed: %v", containerID, err)
		return err
	}

	// start container
	if err = f.StartContainer(ctx); err != nil {
		ctx.Logger.Errorf("start container %s failed: %v", containerID, err)
		return err
	}

	// get container info
	info, err := f.RuntimeClient.ContainerInfo(containerID)
	if err != nil {
		ctx.Logger.Errorf("get container %s failed: %s", containerID, err)
		return err
	}

	// setup container network
	if err = f.NetworkManager.SetContainerNet(info.Pid); err != nil {
		ctx.Logger.Errorf("set container %s pid %d network failed: %s", containerID, info.Pid, err)
		return err
	}

	f.ContainerManager.ContainerMap.UpdateContainerStreamMode(containerID, false)
	f.ContainerManager.ContainerMap.UpdateContainerPid(containerID, info.Pid)
	return nil
}

func (f *Funclet) prepareConfigData(containerID string, cp *file.ContainerPaths) error {
	if err := f.appendContainerHosts(containerID); err != nil {
		f.logger.Errorf("generate etc hosts failed : %v", err)
		return err
	}
	confPath := filepath.Join(cp.RunnerSpecPath, runner.SpecConfig)
	c := &runner.RunnerConfig{
		HostName:          containerID,
		HostsPath:         cp.SpecConfigPath + "/hosts",
		ConfigPath:        cp.DataConfigPath,
		CodePath:          cp.DataCodePath,
		TmpPath:           cp.RunnerTmpPath,
		RuntimePath:       cp.DataRuntimePath,
		RuntimeSocketPath: fmt.Sprintf(f.Options.RunnerSpecOption.RuntimeSocketPath, containerID),
		RuncConfigPath:    confPath,
	}
	if err := os.MkdirAll(c.RuntimeSocketPath, os.ModePerm); err != nil {
		f.logger.Errorf("mkdir socket path %s , err %s", c.RuntimeSocketPath, err)
		return err
	}
	spec := f.RunnerManager.GenerateRunnerConfig(c)
	data, err := json.MarshalIndent(spec, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(confPath, data, 0666)
}

func (f *Funclet) StartContainer(ctx *funcletCtx.Context) error {
	containerID := ctx.Container.ContainerID
	paths, _ := f.PathManager.GetPaths(containerID)
	std, err := f.initializeIO(containerID)
	if err != nil {
		ctx.Logger.Errorf("prepare container %s stdio error: %s", containerID, err)
		return fmt.Errorf("prepare container %s stdio error: %s", containerID, err)
	}
	cr := runtimeapi.CreateContainerRequest{
		ID:        containerID,
		Bundle:    paths.RunnerSpecPath,
		PidFile:   filepath.Join(paths.RunnerSpecPath, "runner.pid"),
		Detach:    true,
		WithStdio: true,
		Stdio: &runtimeapi.ContainerStdio{
			Stdout: std.stdout,
			Stderr: std.stderr,
		},
	}
	return f.RuntimeClient.StartContainer(&cr)
}

func (f *Funclet) appendContainerHosts(containerID string) error {
	paths, err := f.PathManager.GetPaths(containerID)
	if err != nil {
		return err
	}
	path := paths.SpecConfigPath + "/hosts"
	return runner.AppendHostsFile(path, containerID)
}

func (f *Funclet) SetNetwork(pid int) error {
	if err := f.NetworkManager.SetContainerNet(pid); err != nil {
		return fmt.Errorf("set container net failed: %s", err)
	}

	return nil
}

func (f *Funclet) initializeIO(containerID string) (std *stdio, err error) {
	var pr1, pr2 io.ReadCloser
	std = &stdio{}
	pr1, std.stdout, err = os.Pipe()
	if err != nil {
		return nil, err
	}
	logger1 := logs.NewLogger().WithField("container", containerID).WithField("type", "stdout")
	go ReaderToLog(pr1, logger1)

	pr2, std.stderr, err = os.Pipe()
	if err != nil {
		return nil, err
	}
	logger2 := logs.NewLogger().WithField("container", containerID).WithField("type", "stderr")
	go ReaderToLog(pr2, logger2)
	return
}
