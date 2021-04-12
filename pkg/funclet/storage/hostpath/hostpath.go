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

// Package hostpath
package hostpath

import (
	"fmt"
	"os"
	"strings"

	"github.com/baidu/easyfaas/pkg/util/mount"

	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/funclet/device/utils"
	"github.com/baidu/easyfaas/pkg/funclet/storage"
	"github.com/baidu/easyfaas/pkg/util/bytefmt"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

type storageTypeHostPath struct{}

func init() {
	storage.RegisterStorage(api.TmpStorageTypeHostPath, &storageTypeHostPath{})
}

func (s *storageTypeHostPath) Allocate(fpath string, size uint64) (storagePath string, err error) {
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		return "", fmt.Errorf("tmp storage path %s does not exist", fpath)
	}

	availableSize := utils.GetAvailableSize(fpath)
	logs.Infof("tmp storage need size %s, available size %s", bytefmt.ByteSize(size), bytefmt.ByteSize(availableSize))

	if size > availableSize {
		return "", fmt.Errorf("tmp storage size %s is insufficient, at least set to %s", bytefmt.ByteSize(availableSize), bytefmt.ByteSize(size))
	}

	return fpath, nil
}

func (s *storageTypeHostPath) Recycle() {
	return
}

func (s *storageTypeHostPath) PrepareXfs(path string) (err error) {
	isXfs, err := utils.IsXfs(path)
	if err != nil {
		return fmt.Errorf("check xfs path %s failed: %s", path, err)
	}
	if !isXfs {
		return fmt.Errorf("xfs path %s is not an xfs filesystem", path)
	}
	return nil
}

func (s *storageTypeHostPath) MountProjQuota(sourcePath string, targetPath string) (err error) {
	// can't set the mount option of pvc in container
	// mount option should set through the configuration of storage class

	// check the mount option
	mountInfo, err := mount.GetMountInfo(sourcePath)
	if err != nil {
		return fmt.Errorf("get tmp storage path %s mountpoint failed: %s", sourcePath, err)
	}
	if mountInfo.Fstype != "xfs" {
		return fmt.Errorf("fstype of mountpoint %s is not xfs, is %s", sourcePath, mountInfo.Fstype)
	}
	if !strings.Contains(mountInfo.VfsOpts, "pquota") && !strings.Contains(mountInfo.VfsOpts, "prjquota") {
		return fmt.Errorf("tmp storage path %s was not mounted with pquota nor prjquota", sourcePath)
	}
	// bind mount disk
	if err := mount.BindMount(sourcePath, targetPath); err != nil {
		return fmt.Errorf("tmp storage path %s mount runner tmp failed: %s", sourcePath, err)
	}
	return nil
}
