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

package mount

import (
	"syscall"
	"time"
)

func MakeShared(mountPoint string) error {
	if err := syscall.Mount("", mountPoint, "", uintptr(syscall.MS_SHARED), ""); err != nil {
		return err
	}
	return nil
}

func BindMount(source string, target string) error {
	if err := syscall.Mount(source, target, "", uintptr(syscall.MS_BIND)|uintptr(syscall.MS_RDONLY), ""); err != nil {
		return err
	}
	return nil
}

func MountProjectQuota(source string, target string) error {
	if err := syscall.Mount(source, target, "xfs", syscall.MS_MGC_VAL, "prjquota"); err != nil {
		return err
	}
	return nil
}

func GetMounts() ([]*Info, error) {
	return ParseMountTable()
}

func GetMountInfo(mountpoint string) (*Info, error) {
	entries, err := ParseMountTable()
	if err != nil {
		return nil, err
	}

	// Search the table for the mountpoint
	for _, e := range entries {
		if e.Mountpoint == mountpoint {
			return e, nil
		}
	}
	return nil, MountInfoNotFound
}

func Mounted(mountpoint string) (bool, error) {
	entries, err := ParseMountTable()
	if err != nil {
		return false, err
	}

	// Search the table for the mountpoint
	for _, e := range entries {
		if e.Mountpoint == mountpoint {
			return true, nil
		}
	}
	return false, nil
}

func Unmount(target string) error {
	mounted, err := Mounted(target)
	if err != nil {
		return err
	}
	if !mounted {
		return NoNeedUnmount
	}
	return ForceUnmount(target)
}

func ForceUnmount(target string) (err error) {
	for i := 0; i < 10; i++ {
		if err = syscall.Unmount(target, 0); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return
}
