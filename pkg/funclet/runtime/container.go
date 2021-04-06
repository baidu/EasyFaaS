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
package runtime

import (
	"github.com/baidu/openless/pkg/funclet/runtime/api"
	runtimeErr "github.com/baidu/openless/pkg/funclet/runtime/error"
	"github.com/baidu/openless/pkg/funclet/runtime/runc"
	"github.com/baidu/openless/pkg/util/logs"
)

func NewContainerRuntime(runtimeType string, cmd string, logger *logs.Logger) (cm api.ContainerManager, err error) {
	if runtimeType != api.RuntimeTypeRunc {
		return nil, runtimeErr.ErrUnsupportedContainerRuntime{runtimeType}
	}
	cm = runc.NewContainerRuntime(cmd, logger)
	return cm, nil
}
