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

	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

// PathManagerInterface
type PathManagerInterface interface {
	GeneratePaths(id string) (cp *ContainerPaths, err error)
	GetPaths(id string) (cp *ContainerPaths, err error)
	OutdatePaths(id string) error
	OutdatePathsByName(pathName string) error
	RemoveByPathName(name string) error
	GetContainerID(pName string) (cName string, err error)
}

// PathManager
type PathManager struct {
	config         *PathConfig
	containersPath *ContainerPathsMap

	unloadCh chan string
	clearCh  chan string
	logger   *logs.Logger
}

// NewPathManager
func NewPathManager(c *PathConfig, unloadCh chan string, clearCh chan string) *PathManager {
	// TODO: Need to recycle resource(spec/data) in non-container environment.
	m := &PathManager{
		config:         c,
		containersPath: InitContainerPathsMap(),
		unloadCh:       unloadCh,
		clearCh:        clearCh,
		logger:         logs.NewLogger().WithField("module", "PathManager").WithField(api.AppNameKey, "funclet"),
	}
	go m.removeTask()
	return m
}

// TODO: Need to get container list from runc in non-container environment.
func InitContainerPathsMap() *ContainerPathsMap {
	cpm := new(ContainerPathsMap)
	// TODO: 100 is a magic num
	cpm.Map = make(map[string]*ContainerPaths, 100)
	return cpm
}

func (m *PathManager) removeTask() {
	defer func() {
		if err := recover(); err != nil {
			m.logger.Errorf("remove task panic : %s", err)
		}
	}()
	var emptyT int
	for {
		select {
		case path, ok := <-m.clearCh:
			if !ok {
				m.logger.Infof("clear chan has been closed")
				break
			}
			emptyT = 0
			m.logger.V(8).Infof("remove by path name %s start", path)
			err := m.RemoveByPathName(path)
			if err != nil {
				// unexpected error
				m.logger.Errorf("remove by path name %s failed: %s", path, err)
			} else {
				m.logger.V(8).Infof("remove by path name %s success", path)
			}
		default:
			time.Sleep(time.Duration(emptyT) * time.Second)
			emptyT++
		}

	}
}
