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
	"time"

	"github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/util/logs"
	"github.com/baidu/openless/pkg/util/mount"
)

// MountManagerInterface
type MountManagerInterface interface {
	FindDevice(path string) (*mount.Info, error)
	MakeShared(path string) error
	BindMount(pairs []*MountPair, atomic bool) error
	Unmount(path []string, ignoreError bool) error
	UnmountAllByPathName(name string) error
}

// MountManager
type MountManager struct {
	MountInfo []*mount.Info

	config *PathConfig

	unloadCh chan string
	clearCh  chan string

	logger *logs.Logger
}

// NewMountManager
func NewMountManager(c *PathConfig, unloadCh chan string, clearCh chan string) *MountManager {
	mountInfo, err := mount.ParseMountTable()
	if err != nil {
		mountInfo = []*mount.Info{}
	}
	m := &MountManager{
		MountInfo: mountInfo,
		config:    c,
		unloadCh:  unloadCh,
		clearCh:   clearCh,
		logger:    logs.NewLogger().WithField("module", "MountManager").WithField(api.AppNameKey, "funclet"),
	}
	go m.unmountTask()
	return m
}

func (m *MountManager) unmountTask() {
	defer func() {
		if err := recover(); err != nil {
			m.logger.Errorf("unmount task panic : %s", err)
		}
	}()
	var emptyT int
	for {
		select {
		case path, ok := <-m.unloadCh:
			if !ok {
				m.logger.Infof("unmount chan has been closed")
				break
			}
			if path == "" {
				continue
			}
			emptyT = 0
			m.logger.V(8).Infof("unmount by path name %s start", path)
			err := m.UnmountAllByPathName(path)
			if err != nil {
				m.logger.Errorf("unmount by path name %s failed: %s", path, err)
				m.unloadCh <- path
			} else {
				m.logger.V(8).Infof("unmount by path name %s success", path)
				m.clearCh <- path
			}
		default:
			time.Sleep(time.Duration(emptyT) * time.Second)
			emptyT++
		}

	}
}
