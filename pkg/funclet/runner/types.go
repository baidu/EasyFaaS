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

// Package runner
package runner

import (
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/baidu/easyfaas/pkg/funclet/runtime/api"
)

type RunnerSpec = specs.Spec

type RunnerConfig struct {
	HostName          string
	HostsPath         string
	ConfigPath        string
	CodePath          string
	RuntimePath       string
	TmpPath           string
	RuncConfigPath    string
	RuntimeSocketPath string
	ResourceConfig    *api.ResourceConfig
}
