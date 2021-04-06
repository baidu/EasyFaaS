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

package file

import (
	"sync"
)

type ContainerPathsMap struct {
	Map  map[string]*ContainerPaths
	lock sync.RWMutex
}

type ContainerPaths struct {
	PathName string

	RunnerDataPath    string
	RunnerSpecPath    string
	RunnerTmpPath     string
	CodeWorkspacePath string

	SpecConfigPath  string
	DataConfigPath  string
	DataCodePath    string
	DataRuntimePath string
}

type PathConfig struct {
	RunningMode       string
	RunnerDataPath    string
	RunnerSpecPath    string
	RunnerTmpPath     string
	CodeWorkspacePath string
	EtcPath           string
	CodePath          string
	ConfPath          string
	RuntimePath       string
}

type MountPair struct {
	Source string
	Target string
}
