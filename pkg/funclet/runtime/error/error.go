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

// Package runtime
package error

import "fmt"

type ErrUnsupportedContainerRuntime struct {
	ContainerRuntimeName string
}

func (err ErrUnsupportedContainerRuntime) Error() string {
	return fmt.Sprintf("%s: unsupported method", err.ContainerRuntimeName)
}

type ErrInsufficientResources struct {
	Has bool
	Err error
}

func (err ErrInsufficientResources) Error() string {
	return fmt.Sprintf("check container resources failed: [isSufficient %t] [Err %s]", err.Has, err.Err)
}

type ErrCgroupNotExist struct {
	ID string
}

func (err ErrCgroupNotExist) Error() string {
	return fmt.Sprintf("%s: cgroup not exist", err.ID)
}

type GetContainerInfoError struct {
	ID  string
	Err error
}

func (err GetContainerInfoError) Error() string {
	return fmt.Sprintf("get container %s info failed: %s", err.ID, err.Err)
}
