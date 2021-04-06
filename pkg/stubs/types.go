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

package stubs

import (
	"github.com/baidu/openless/pkg/api"
)

var (
	runtimesMap = map[string]*api.RuntimeConfiguration{
		"nodejs6.11": {
			Name: "nodejs6.11",
			Bin:  "/bin/bash",
			Path: "/var/faas/runtime/node-v6.11.3-linux-x64",
			Args: []string{"/var/runtime/bootstrap"},
		},
		"nodejs8.5": {
			Name: "nodejs8.5",
			Bin:  "/bin/bash",
			Path: "/var/faas/runtime/node-v8.5.0-linux-x64",
			Args: []string{"/var/runtime/bootstrap"},
		},
		"nodejs10": {
			Name: "nodejs10",
			Bin:  "/bin/bash",
			Path: "/var/faas/runtime/node-v10.15.3-linux-x64",
			Args: []string{"/var/runtime/bootstrap"},
		},
		"lua5.3": {
			Name: "lua5.3",
			Bin:  "/bin/bash",
			Path: "/var/faas/runtime/lua-v5.3.5-x86_64-linux-gnu",
			Args: []string{"/var/runtime/script/kunruntime.lua"},
		},
	}
)

type CreateFunctionArgs struct {
	Version            string
	Description        string
	Runtime            string
	Timeout            int64
	MemorySize         int64
	Handler            string
	PodConcurrentQuota uint64
	Code               string
}
