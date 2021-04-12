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

// Package loop
package loop

import (
	"fmt"
	"strings"

	"github.com/baidu/easyfaas/pkg/funclet/device/utils"

	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/funclet/device/loop"
	"github.com/baidu/easyfaas/pkg/funclet/storage"
	"github.com/baidu/easyfaas/pkg/util/file"
	"github.com/baidu/easyfaas/pkg/util/logs"
	"github.com/baidu/easyfaas/pkg/util/mount"
)

func init() {
	storage.RegisterStorage(api.TmpStorageTypeLoop, &storageTypeLoop{})
}

type storageTypeLoop struct{}

func (s *storageTypeLoop) Allocate(fpath string, size uint64) (storagePath string, err error) {

	if err := file.CreateFile(fpath, int64(size)); err != nil {
		logs.Errorf("create tmp storage failed: %s", err)
		return "", err
	}
	logs.Infof("finish prepare tmp storage")

	// generate loop device for tmp
	logs.Infof("start attach loop device for tmp storage")
	tmpDevice, err := loop.Attach(fpath, 0, false)
	if err != nil {
		logs.Errorf("attach tmp storage failed:", err)
		return "", err
	}
	storagePath = tmpDevice.Path()
	return storagePath, nil
}

func (s *storageTypeLoop) Recycle() {
	go recycleLoopDevice()
}

func (s *storageTypeLoop) PrepareXfs(path string) (err error) {
	// mkfs.xfs
	if err := utils.MkfsXfs(path); err != nil {
		return fmt.Errorf("mkfs xfs failed: %s", err)
	}
	return nil
}

func (s *storageTypeLoop) MountProjQuota(sourcePath string, targetPath string) (err error) {
	// mount with project quota
	if err := mount.MountProjectQuota(sourcePath, targetPath); err != nil {
		return fmt.Errorf("mount runner tmp source %s target %s failed: %s", sourcePath, targetPath, err)
	}
	return nil
}

func recycleLoopDevice() {
	loopDeviceMap, err := loop.GetLoopDeviceMap()
	if err != nil {
		return
	}
	for filePath, devices := range loopDeviceMap {
		for _, device := range devices {
			if strings.Contains(filePath, "faas-tmp") || device.IsDeleted {
				if err := device.Detach(); err != nil {
					continue
				}
			}
		}
	}
	return
}
