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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/baidu/easyfaas/pkg/api"
)

var (
	PathsExists    = errors.New("container paths exists")
	PathsNotExists = errors.New("container paths not exists")
)

func (m *PathManager) GeneratePaths(id string) (cp *ContainerPaths, err error) {
	cp, err = m.newContainerPaths(id)
	if err != nil {
		return nil, err
	}

	pathName := fmt.Sprintf("%s-%d", id, time.Now().UnixNano())

	paths := make([]string, 0)
	etcPath, _ := m.config.GetPathByName(ETC, pathName)
	paths = append(paths, etcPath)

	confPath, _ := m.config.GetPathByName(CONFIG, pathName)
	paths = append(paths, confPath)

	taskPath, _ := m.config.GetPathByName(CODE, pathName)
	paths = append(paths, taskPath)

	runtimePath, _ := m.config.GetPathByName(RUNTIME, pathName)
	paths = append(paths, runtimePath)

	tmpPath, _ := m.config.GetPathByName(TMPDIR, pathName)
	paths = append(paths, tmpPath)

	if m.config.RunningMode == api.RunningModeIDE {
		workspacePath, _ := m.config.GetPathByName(WORKSPACE, pathName)
		paths = append(paths, workspacePath)
		cp.CodeWorkspacePath = workspacePath
	}

	for _, p := range paths {
		path := p
		if err := os.MkdirAll(path, 0755); err != nil {
			m.logger.Errorf("init container %s dir %s failed: %s", id, path, err)
			m.OutdatePaths(id)
			return nil, fmt.Errorf("init container %s dir %s failed: %s", id, path, err)
		}
	}
	cp.PathName = pathName

	cp.RunnerSpecPath, _ = m.config.GetPathByName(SPEC, pathName)
	cp.RunnerDataPath, _ = m.config.GetPathByName(DATA, pathName)
	cp.SpecConfigPath = etcPath
	cp.DataConfigPath = confPath
	cp.DataCodePath = taskPath
	cp.DataRuntimePath = runtimePath
	cp.RunnerTmpPath = tmpPath

	return cp, nil
}

func (m *PathManager) GetPaths(id string) (cp *ContainerPaths, err error) {
	m.containersPath.lock.Lock()
	defer m.containersPath.lock.Unlock()
	if m.containersPath.Map[id] == nil {
		return nil, PathsNotExists
	}
	return m.containersPath.Map[id], nil
}

func (m *PathManager) OutdatePaths(id string) error {
	m.containersPath.lock.Lock()
	defer m.containersPath.lock.Unlock()
	p := m.containersPath.Map[id]
	if p == nil {
		return PathsNotExists
	}
	m.unloadCh <- p.PathName
	m.containersPath.Map[id] = nil
	return nil
}

func (m *PathManager) OutdatePathsByName(pathName string) error {
	m.unloadCh <- pathName
	return nil
}

func (m *PathManager) GetContainerID(pName string) (cName string, err error) {
	pos := strings.LastIndex(pName, "-")
	if pos < 0 {
		return "", fmt.Errorf("invalid path name %s", pName)
	}
	return pName[0:pos], nil
}

func (m *PathManager) RemoveByPathName(name string) error {
	paths := make([]string, 0)
	confPath, _ := m.config.GetPathByName(CONFIG, name)
	paths = append(paths, confPath)

	taskPath, _ := m.config.GetPathByName(CODE, name)
	paths = append(paths, taskPath)

	runtimePath, _ := m.config.GetPathByName(RUNTIME, name)
	paths = append(paths, runtimePath)

	for _, p := range paths {
		path := p
		dir, err := ioutil.ReadDir(p)
		if err != nil {
			m.logger.Errorf("read dir %s failed", p)
		}
		if len(dir) != 0 {
			err = fmt.Errorf("path %s not empty, something wrong ... ", path)
			m.logger.Errorf("remove by path name %s failed: %s", p, err)
			return err
		}
	}

	specPath, _ := m.config.GetPathByName(SPEC, name)
	if err := os.RemoveAll(specPath); err != nil {
		m.logger.Errorf("remove path %s failed: %s", specPath, err)
		return err
	}
	dataPath, _ := m.config.GetPathByName(DATA, name)
	if err := os.RemoveAll(dataPath); err != nil {
		m.logger.Errorf("remove path %s failed: %s", dataPath, err)
		return err
	}
	return nil
}

func (m *PathManager) newContainerPaths(id string) (cp *ContainerPaths, err error) {
	m.containersPath.lock.Lock()
	defer m.containersPath.lock.Unlock()
	if m.containersPath.Map[id] != nil {
		return nil, PathsExists
	}
	cp = &ContainerPaths{}
	m.containersPath.Map[id] = cp
	return cp, nil
}
