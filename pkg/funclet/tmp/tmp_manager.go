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

// Package tmp
package tmp

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/baidu/easyfaas/pkg/funclet/storage"

	"github.com/baidu/easyfaas/pkg/funclet/quota"

	"github.com/baidu/easyfaas/pkg/util/bytefmt"

	"github.com/baidu/easyfaas/pkg/api"

	"github.com/baidu/easyfaas/pkg/util/logs"

	_ "github.com/baidu/easyfaas/pkg/funclet/storage/hostpath"
	_ "github.com/baidu/easyfaas/pkg/funclet/storage/loop"
)

var (
	RunnersTmpPath    = "/var/faas/runner-tmp"
	CodeWorkspacePath = "/var/faas/code-workspace"
	StorageRatio      = 1.5
)

// TmpManagerInterface
type TmpManagerInterface interface {
	GetTmpStorage(name string) (path string, err error)
	RemoveTmpStorage(name string) (err error)
	SnapshotTmpPaths() (paths *[]string, err error)
	ListTmpPaths() (paths *[]string, err error)
}

type TmpManager struct {
	options      *TmpStorageOption
	containerNum int
	quota        quota.QuotaCtrl
	logger       *logs.Logger
}

func NewTmpManager(o *TmpStorageOption, containerNum int) (manager *TmpManager, err error) {
	manager = &TmpManager{
		options:      o,
		containerNum: containerNum,
		logger:       logs.NewLogger().WithField("module", "TmpManager").WithField(api.AppNameKey, "funclet"),
	}
	manager.logger.Infof("start init tmp manager")
	defer manager.logger.Infof("finish init tmp manager")

	// validate and generate storage path
	fpath, err := manager.getStoragePath()
	if err != nil {
		err := fmt.Errorf("check storage path failed: %s", err)
		manager.logger.Error(err.Error())
		return nil, err
	}

	// get required storage size
	requiredSize, err := manager.requiredStorageSize()
	if err != nil {
		err := fmt.Errorf("get required storage size failed: %s", err)
		manager.logger.Error(err.Error())
		return nil, err
	}

	// get storage by type
	ts, err := storage.GetStorage(o.TmpStorageType)
	if err != nil {
		manager.logger.Errorf("init tmp storage type %s failed: %s", o.TmpStorageType, err)
		return nil, err
	}

	// allocate storage
	storagePath, err := ts.Allocate(fpath, requiredSize)
	if err != nil {
		manager.logger.Errorf("alloc storage failed: %s", err)
		return nil, err
	}

	// init quota controller
	manager.logger.Infof("start init quota controller")
	quota, err := quota.InitQuotaCtrl(storagePath, o.RunnerTmpSize, RunnersTmpPath, o.TmpStorageType, manager.logger)
	if err != nil {
		manager.logger.Errorf("init quota controller failed:", err)
		return nil, err
	}
	manager.quota = quota

	ts.Recycle()

	return manager, nil
}

func (m *TmpManager) getStoragePath() (path string, err error) {
	// prepare tmp storage
	if err := os.MkdirAll(m.options.TmpStoragePath, 0777); err != nil {
		return "", err
	}
	if m.options.TmpStorageType == api.TmpStorageTypeLoop {
		path = filepath.Join(m.options.TmpStoragePath, m.options.TmpStorageName)
	} else {
		path = m.options.TmpStoragePath
	}
	return
}

func (m *TmpManager) requiredStorageSize() (requiredStorage uint64, err error) {
	runnerBytes, err := bytefmt.ToBytes(m.options.RunnerTmpSize)
	if err != nil {
		return 0, fmt.Errorf("invalid runner tmp size %s", m.options.RunnerTmpSize)
	}

	needStorageSize := StorageRatio * float64(runnerBytes) * float64(m.containerNum)
	return uint64(needStorageSize), nil
}

func (m *TmpManager) GetTmpStorage(name string) (path string, err error) {
	path = filepath.Join(RunnersTmpPath, name)
	if err := m.quota.AllocTmpDevice(path); err != nil {
		return "", err
	}
	return path, nil
}

func (m *TmpManager) RemoveTmpStorage(name string) (err error) {
	path := filepath.Join(RunnersTmpPath, name)
	return m.quota.FreeTmpDevice(path)
}

func (m *TmpManager) SnapshotTmpPaths() (paths *[]string, err error) {
	return m.quota.SnapshotProjectPaths()
}

func (m *TmpManager) ListTmpPaths() (paths *[]string, err error) {
	pathArr := make([]string, 0)
	files, err := ioutil.ReadDir(RunnersTmpPath)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		pathArr = append(pathArr, f.Name())
	}
	return &pathArr, nil
}
