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

// Package runc
package runc

import (
	"github.com/baidu/easyfaas/pkg/funclet/runtime/api"
	runtimeErr "github.com/baidu/easyfaas/pkg/funclet/runtime/error"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

type runcContainerRuntime struct {
	runtimeCtrl RuncCtl
	Logger      *logs.Logger
}

func NewContainerRuntime(cmd string, logger *logs.Logger) api.ContainerManager {
	rc := RuncConfig{
		Command: cmd,
		Logger:  logger,
	}
	return &runcContainerRuntime{
		runtimeCtrl: &rc,
		Logger:      logger,
	}
}

func (r *runcContainerRuntime) Name() string {
	return api.RuntimeTypeRunc
}

func (r *runcContainerRuntime) StartContainer(request *api.CreateContainerRequest) error {
	opts := CreateOpts{
		ID:      request.ID,
		Bundle:  request.Bundle,
		PidFile: request.PidFile,
		Detach:  request.Detach,
	}
	if request.WithStdio {
		return r.runtimeCtrl.RunWithStdio(&opts, request.Stdio.Stdout, request.Stdio.Stderr)
	}
	return r.runtimeCtrl.Run(&opts)
}

func (r *runcContainerRuntime) RemoveContainer(ID string, force bool) error {
	return r.runtimeCtrl.Delete(ID, force)
}

func (r *runcContainerRuntime) KillContainer(ID string, signal string, all bool) error {
	return r.runtimeCtrl.Kill(ID, signal, all)
}

func (r *runcContainerRuntime) PauseContainer(ID string) error {
	return r.runtimeCtrl.Pause(ID)
}

func (r *runcContainerRuntime) ResumeContainer(ID string) error {
	return r.runtimeCtrl.Resume(ID)
}

func (r *runcContainerRuntime) ListContainers() (list []*api.Container, err error) {
	return r.runtimeCtrl.List()
}

func (r *runcContainerRuntime) ContainerInfo(ID string) (container *api.Container, err error) {
	container, stateErr := r.runtimeCtrl.State(ID)
	if stateErr != nil {
		err = runtimeErr.GetContainerInfoError{
			ID:  ID,
			Err: stateErr,
		}
	}
	return
}

func (r *runcContainerRuntime) UpdateContainer(ID string, request *api.UpdateContainerRequest) error {
	uo := UpdateResourceOpts{
		ID:       ID,
		Memory:   request.Memory,
		CPUQuota: request.CPUQuota,
	}
	return r.runtimeCtrl.UpdateResource(&uo)
}
