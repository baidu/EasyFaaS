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

package runner

import (
	"fmt"
	"io"
	"os"

	"github.com/baidu/easyfaas/pkg/util/logs"
)

func PrepareConfigDir(containerID string, path string) error {
	hostsPath := path + "/hosts"
	if err := GenerateHostsFile(hostsPath); err != nil {
		logs.Errorf("generate container %s hosts file failed", containerID)
		return fmt.Errorf("generate container %s hosts file failed", containerID)
	}
	return nil
}

func GenerateHostsFile(path string) error {
	src := "/etc/hosts"
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.OpenFile(src, os.O_RDONLY, 0644)
	if err != nil {
		logs.Errorf("open /etc/hosts failed: %+v", err)
		return err
	}
	defer source.Close()

	destination, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		logs.Errorf("create runner etc hosts file (%s) failed: %+v", path, err)
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		logs.Errorf("copy etc/hosts failed: %+v", err)
		return err
	}
	return nil
}

func AppendHostsFile(path, containerID string) error {
	hostsFd, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer hostsFd.Close()
	if _, err := hostsFd.WriteString(fmt.Sprintf("127.0.0.1  %s", containerID)); err != nil {
		logs.Errorf("write runner etc hosts (%s) failed: %+v", path, err)
		return err
	}
	return nil
}
