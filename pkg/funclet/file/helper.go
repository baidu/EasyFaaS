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
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
)

type pathType string

const (
	SPEC pathType = "spec"
	DATA pathType = "data"

	ETC       pathType = "etc"
	CONFIG    pathType = "config"
	CODE      pathType = "code"
	RUNTIME   pathType = "runtime"
	TMPDIR    pathType = "tmp"
	WORKSPACE pathType = "workspace"
)

var (
	UnknownPathType = errors.New("unknown path type")
)

func (c *PathConfig) GetPathByName(t pathType, n string) (path string, err error) {
	switch t {
	case SPEC:
		path = filepath.Join(c.RunnerSpecPath, n)
	case DATA:
		path = filepath.Join(c.RunnerDataPath, n)
	case ETC:
		path = fmt.Sprintf(c.EtcPath, n)
	case CONFIG:
		path = fmt.Sprintf(c.ConfPath, n)
	case CODE:
		path = fmt.Sprintf(c.CodePath, n)
	case RUNTIME:
		path = fmt.Sprintf(c.RuntimePath, n)
	case TMPDIR:
		path = filepath.Join(c.RunnerTmpPath, n)
	case WORKSPACE:
		path = filepath.Join(c.CodeWorkspacePath, n)
	default:
		err = UnknownPathType
	}
	return
}

func (c *PathConfig) GetPathMapByName(n string) (pathMap map[pathType]string) {
	pathMap = make(map[pathType]string, 6)
	pathMap[SPEC], _ = c.GetPathByName(SPEC, n)
	pathMap[DATA], _ = c.GetPathByName(DATA, n)
	pathMap[ETC], _ = c.GetPathByName(ETC, n)
	pathMap[CONFIG], _ = c.GetPathByName(CONFIG, n)
	pathMap[CODE], _ = c.GetPathByName(CODE, n)
	pathMap[RUNTIME], _ = c.GetPathByName(RUNTIME, n)
	pathMap[TMPDIR], _ = c.GetPathByName(TMPDIR, n)
	pathMap[WORKSPACE], _ = c.GetPathByName(WORKSPACE, n)
	return
}
