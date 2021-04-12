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

package cgroup

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/baidu/easyfaas/pkg/api"
)

func GetFreezerStats(path string) (api.FreezerState, error) {
	state, err := readFile(path, "freezer.state")
	if err != nil {
		return "", err
	}
	return api.FreezerState(state), nil
}

func readFile(dir, file string) (string, error) {
	path := filepath.Join(dir, file)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return string(api.Undefined), nil
	}
	data, err := ioutil.ReadFile(path)
	return strings.TrimSpace(string(data)), err
}
