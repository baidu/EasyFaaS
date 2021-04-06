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

package runc

type RuncContainerStatus = string

const (
	ContainerStatusNotExist RuncContainerStatus = "not exists"
	// Created is the status that denotes the container exists but has not been run yet.
	ContainerStatusCreated RuncContainerStatus = "created"
	// Running is the status that denotes the container exists and is running.
	ContainerStatusRunning RuncContainerStatus = "running"
	// Pausing is the status that denotes the container exists, it is in the process of being paused.
	ContainerStatusPausing RuncContainerStatus = "pausing"
	// Paused is the status that denotes the container exists, but all its processes are paused.
	ContainerStatusPaused RuncContainerStatus = "paused"
	// Stopped is the status that denotes the container does not have a created or running process.
	ContainerStatusStopped RuncContainerStatus = "stopped"
)
