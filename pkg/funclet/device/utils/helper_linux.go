// +build linux

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


// Package utils
package utils

import (
	"syscall"

	funletCmd "github.com/baidu/openless/pkg/funclet/command"
)

func MkfsXfs(fileName string) error {
	_, err := funletCmd.CommandOutput("mkfs.xfs", fileName)
	if err != nil {
		return err
	}
	return nil
}

func IsXfs(xfsPath string) (bool, error) {
	var stat syscall.Statfs_t
	syscall.Statfs(xfsPath, &stat)
	if getFSType(stat.Type) == "XFS" {
		return true, nil
	}
	return false, nil
}

func GetDevice(filePath string) (uint64, error) {
	stat := syscall.Stat_t{}
	err := syscall.Stat(filePath, &stat)
	if err != nil {
		return 0, err
	}
	return stat.Rdev, nil
}
