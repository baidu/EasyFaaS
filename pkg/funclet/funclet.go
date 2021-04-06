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
	"fmt"
	"os"
	"time"

	"github.com/baidu/openless/pkg/funclet/runtime"

	"github.com/baidu/openless/pkg/funclet/runner"

	"github.com/baidu/openless/pkg/funclet/tmp"

	"github.com/baidu/openless/cmd/funclet/options"
	"github.com/baidu/openless/pkg/funclet/file"

	"github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/funclet/code"
	funcletCtx "github.com/baidu/openless/pkg/funclet/context"
	"github.com/baidu/openless/pkg/funclet/network"
	"github.com/baidu/openless/pkg/util/logs"
)

// Funclet
type Funclet struct {
	PodName          string
	Options          *options.FuncletOptions
	logger           *logs.Logger
	RuntimeClient    runtime.RuntimeManagerInterface
	CodeManager      code.ManagerInterface
	RunnerManager    runner.RunnerManagerInterface
	MountManager     file.MountManagerInterface
	PathManager      file.PathManagerInterface
	NetworkManager   network.NetworkManagerInterface
	TmpManager       tmp.TmpManagerInterface
	ContainerManager *ContainerManager

	HandlingChanMap *HandlingChanMap
}

func InitFunclet(o *options.FuncletOptions, stopCh <-chan struct{}) (f *Funclet, err error) {
	logger := logs.NewLogger()
	podName := os.Getenv("MY_POD_NAME")
	if podName == "" {
		logger.Warnf("pod name is empty, use 'funclet' as default")
		podName = "funclet"
	}

	p := runtime.RuntimeManagerParameters{
		RuntimeCmd:   o.RuntimeCmd,
		ContainerNum: o.ContainerNum,
		Option:       o.ResourceOption,
		Logger:       logger,
	}
	runtimeClient, err := runtime.NewRuntimeManager(&p)
	if err != nil {
		return nil, err
	}

	o.TmpStorageOption.TmpStorageName = fmt.Sprintf("tmp-%s", podName)
	tm, err := tmp.NewTmpManager(o.TmpStorageOption, o.ContainerNum)
	if err != nil {
		return nil, err
	}

	pc := GetPathConfig(o)
	// outdate path task => unmount path task
	unloadCh := make(chan string, 1000000)
	// unmount path task => remove path task
	clearCh := make(chan string, 1000000)

	f = &Funclet{
		PodName:          podName,
		Options:          o,
		RuntimeClient:    runtimeClient,
		RunnerManager:    runner.NewRunnerManager(o.RunnerSpecOption, runtimeClient.GetDefaultResourceConfig(), o.RunningMode),
		CodeManager:      code.NewManager(o.TmpPath),
		MountManager:     file.NewMountManager(pc, unloadCh, clearCh),
		PathManager:      file.NewPathManager(pc, unloadCh, clearCh),
		TmpManager:       tm,
		NetworkManager:   network.NewNetworkManager(),
		ContainerManager: NewContainerManager(podName, o),
		HandlingChanMap:  NewHandlingChanMap(),
		logger:           logger,
	}

	// wait for rootfs
	if err := checkRootfs(60*time.Second, o.RunnerSpecOption.RootfsPath, logger); err != nil {
		f.logger.Errorf("check rootfs failed: %s", err)
		return nil, err
	}

	// setup propagation flag of the directory of runner's data to shared
	if err := f.MountManager.MakeShared(f.Options.RunnerDataPath); err != nil {
		f.logger.Errorf("set mount propagation shared failed: %s", err)
		return nil, err
	}

	// create network bridge
	if err := f.NetworkManager.InitNetwork(f.Options.NetworkOption); err != nil {
		f.logger.Errorf("init network bridge failed: %s", err)
		return nil, err
	}

	// init container map
	f.InitAllContainers()

	go f.RecycleTask(stopCh)

	return f, nil
}

func (f *Funclet) InitAllContainers() {
	for i := 0; i < f.Options.ContainerNum; i++ {
		id := generateContainerID(f.PodName, i)
		if err := f.InitContainerEvent(id); err != nil {
			f.logger.Errorf("init container %s failed: %+v", id, err)
		}
	}
}

// tools to reset node
func (f *Funclet) Reset() (err error) {
	runtimeList, err := f.RuntimeClient.ListContainers()
	if err != nil {
		return err
	}
	if len(runtimeList) == 0 {
		return nil
	}
	for _, item := range runtimeList {
		fCtx := &funcletCtx.Context{Logger: f.logger}
		params := &api.ResetRequest{
			ContainerID: item.ID,
		}
		f.ResetContainerEvent(fCtx, params)
	}

	return nil
}
