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

// Package device
package quota

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
)

func TestAddToFile(t *testing.T) {
	q := &xfsQuotaCtrl{
		dataLock: sync.Mutex{},
		fileLock: sync.Mutex{},
	}
	projectsFile = "./projects"
	f, err := os.Create(projectsFile)
	if err != nil {
		t.Error(err)
		return
	}
	f.Close()
	defer os.Remove(projectsFile)
	projDesc := fmt.Sprintf("%d:%s\n", 10, "/var/faas/runner-tmp/controller-001")
	if err := q.addToFile(projDesc); err != nil {
		t.Error(err)
	}
	data, _ := ioutil.ReadFile(projectsFile)
	t.Logf(string(data))

}

func TestRemoveFromFile(t *testing.T) {
	q := &xfsQuotaCtrl{
		dataLock: sync.Mutex{},
		fileLock: sync.Mutex{},
	}
	projectsFile = "./projects"
	f, err := os.Create(projectsFile)
	if err != nil {
		t.Error(err)
		return
	}
	f.Close()
	defer os.Remove(projectsFile)
	for i := 1; i <= 100; i++ {
		projDesc := fmt.Sprintf("%d:%s\n", i, fmt.Sprintf("/var/faas/runner-tmp/controller-%d", i))
		if err := q.addToFile(projDesc); err != nil {
			t.Error(err)
		}

	}
	if err := q.removeFromFile("/var/faas/runner-tmp/controller-4"); err != nil {
		t.Error(err)
	}
	data, _ := ioutil.ReadFile(projectsFile)
	t.Logf(string(data))
}
