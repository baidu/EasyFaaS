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

package code

import (
	"github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/funclet/context"
	"github.com/baidu/openless/pkg/repository"
	_ "github.com/baidu/openless/pkg/repository/bos"
	"github.com/baidu/openless/pkg/repository/factory"
	_ "github.com/baidu/openless/pkg/repository/filesystem"
	"github.com/baidu/openless/pkg/util/logs"
)

const (
	DriverBos   = "bos"
	DriverLocal = "filesystem"
)

// ManagerInterface
type ManagerInterface interface {
	FetchCode(ctx *context.Context, code *api.CodeStorage) (string, error)
	CheckCode(ctx *context.Context, filename, codeSha256 string) bool
	FindCodeCompeleteTag(path, codeSha256 string) bool
	CreateCodeCompeleteTag(path, codeSha256 string) error
	RemoveCodeCompeleteTag(path, codeSha256 string) error
	FindCode(path, codeSha256 string) bool
	UnzipCode(ctx *context.Context, filename, target string) error
}

// Manager
type Manager struct {
	BasePath string
}

// NewManager
func NewManager(basePath string) *Manager {
	params := map[string]interface{}{
		"basePath": basePath,
	}
	if driver, err := factory.Create(DriverBos, params); err != nil {
		logs.Errorf("install driver %s failed: %s", DriverBos, err)
	} else {
		repository.RegisterStorageDriver(driver)
	}
	if driver, err := factory.Create(DriverLocal, params); err != nil {
		logs.Errorf("install driver %s failed: %s", DriverLocal, err)
	} else {
		repository.RegisterStorageDriver(driver)
	}
	return &Manager{}
}
