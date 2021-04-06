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

// Package api
package api

import (
	"io"
	"time"
)

const RuntimeTypeRunc = "runc"

type ContainerManager interface {
	Name() string
	StartContainer(*CreateContainerRequest) error
	RemoveContainer(ID string, force bool) error
	KillContainer(ID string, signal string, all bool) error
	PauseContainer(ID string) error
	ResumeContainer(ID string) error
	ListContainers() (list []*Container, err error)
	ContainerInfo(ID string) (container *Container, err error)
	UpdateContainer(ID string, request *UpdateContainerRequest) error
}

type CreateContainerRequest struct {
	ID        string
	Bundle    string
	PidFile   string
	Detach    bool
	WithStdio bool
	Stdio     *ContainerStdio
}

type ContainerStdio struct {
	Stdout io.WriteCloser
	Stderr io.WriteCloser
}

type UpdateContainerRequest struct {
	Memory   int64
	CPUQuota int64
}

type Container struct {
	ID          string            `json:"id"`
	Pid         int               `json:"pid"`
	Status      string            `json:"status"`
	Bundle      string            `json:"bundle"`
	Rootfs      string            `json:"rootfs"`
	Created     time.Time         `json:"created"`
	Annotations map[string]string `json:"annotations"`
}
