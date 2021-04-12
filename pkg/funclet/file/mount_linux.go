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

package file

import (
	"syscall"

	"github.com/baidu/easyfaas/pkg/util/mount"
)

func (m *MountManager) MakeShared(path string) error {
	mountInfo, err := m.FindDevice(path)
	if err != nil {
		return err
	}
	if err := mount.MakeShared(mountInfo.Mountpoint); err != nil {
		return err
	}
	return nil
}

func (m *MountManager) BindMount(pairs []*MountPair, atomic bool) error {
	var err error
	for _, p := range pairs {
		pair := p
		if err = mount.BindMount(pair.Source, pair.Target); err != nil {
			m.logger.Errorf("bind mount failed: source [%s] target [%s] error [%s]", pair.Source, pair.Target, err)
			break
		}
	}
	if err != nil && atomic {
		paths := make([]string, 0)
		for _, p := range pairs {
			path := p.Target
			paths = append(paths, path)
		}
		if err = m.Unmount(paths, false); err != nil {
			m.logger.Errorf("unmount failed: paths [%+v] error [%s]", paths, err)
		}
		return err
	}
	return nil
}

func (m *MountManager) Unmount(path []string, ignoreError bool) error {
	for _, item := range path {
		p := item
		m.logger.V(9).Infof("unmount path %s start", p)
		if err := syscall.Unmount(p, 0); err != nil {
			if err == syscall.EINVAL {
				m.logger.V(9).Infof("no need to unmount path %s ", p)
				return nil
			}
			if !ignoreError {
				m.logger.Errorf("unmount path %s err %s", p, err)
				return err
			}
		}
		m.logger.V(9).Infof("unmount path %s stop", p)
	}
	return nil
}

func (m *MountManager) UnmountAllByPathName(name string) error {
	paths := m.config.GetPathMapByName(name)
	mountedPaths, err := m.ParseMounted(paths)
	if err != nil {
		return err
	}
	for _, item := range mountedPaths {
		p := item
		m.logger.V(9).Infof("unmount path %s start", p)
		// simple retry
		for i := 1; i <= 10; i++ {
			if err := mount.Unmount(p); err != nil {
				if err == mount.NoNeedUnmount {
					m.logger.V(9).Infof("no need to unmount path %s ", p)
					break
				} else {
					m.logger.V(9).Infof("unmount path %s err %s", p, err)
					continue
				}
				// check mount
				if mounted, err := mount.Mounted(p); err != nil || !mounted {
					m.logger.V(4).Warnf("syscall unmount %s success, but mountpoints exists, retry times %d", p, i)
					continue
				}
			}
		}
		m.logger.V(9).Infof("unmount path %s stop", p)
	}
	return nil
}

func (m *MountManager) ParseMounted(paths map[pathType]string) (mountedPaths []string, err error) {
	mountInfo, err := mount.GetMounts()
	if err != nil {
		return nil, err
	}
	mountedPaths = make([]string, 0)
	searchMap := make(map[string]struct{})
	for _, item := range paths {
		searchMap[item] = struct{}{}
	}
	for _, e := range mountInfo {
		if _, ok := searchMap[e.Mountpoint]; ok {
			mountedPaths = append(mountedPaths, e.Mountpoint)
		}
	}
	return mountedPaths, nil
}
