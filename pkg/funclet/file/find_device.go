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
	"os"
	"path/filepath"
	"strings"

	"github.com/baidu/openless/pkg/util/mount"
)

// FindDevice
func (d *MountManager) FindDevice(path string) (*mount.Info, error) {
	p := path
	for {
		for _, mountInfo := range d.MountInfo {
			if mountInfo.Mountpoint == p {
				return mountInfo, nil
			}
		}
		p, _ = filepath.Split(p)
		if p != "/" {
			p = strings.TrimRight(p, string(os.PathSeparator))
		}
		if p == "/" || p == "" {
			break
		}
	}
	return nil, fmt.Errorf("Mountpoint not found for %s", path)
}
