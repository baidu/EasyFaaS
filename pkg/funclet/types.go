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

package funclet

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/baidu/easyfaas/pkg/api"
)

type MountInfo struct {
	CodePath    string
	ConfPath    string
	RuntimePath string
}

// Meta Funclet
type Meta struct {
	*api.FunctionConfig
	RuntimePath string
}

type stdio struct {
	stdin  *os.File // not used yet
	stdout *os.File
	stderr *os.File
}

// LoopDeviceContainer Info
type LoopDeviceInfo struct {
	LoopDeviceContainer string
	Target              string
	ReadOnly            bool
	FileType            string
	Device              uint64
}

func (l *LoopDeviceInfo) String() string {
	mode := "ro"
	if !l.ReadOnly {
		mode = "rw"
	}
	return fmt.Sprintf("%s:%s,%d,%s,%s", l.LoopDeviceContainer, l.Target, l.Device, l.FileType, mode)
}

// HandlingChanMap
type HandlingChanMap struct {
	lock *sync.Mutex
	m    map[string]chan string
}

// NewHandlingChanMap
func NewHandlingChanMap() *HandlingChanMap {
	return &HandlingChanMap{
		lock: new(sync.Mutex),
		m:    make(map[string]chan string),
	}
}

// GetChan
func (m *HandlingChanMap) GetChan(key string) (chan string, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	val, ok := m.m[key]
	if !ok {
		m.m[key] = make(chan string, 5)
		return m.m[key], true
	}
	return val, false
}

// CloseChan
func (m *HandlingChanMap) CloseChan(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	val, ok := m.m[key]
	if ok {
		delete(m.m, key)
		close(val)
	}
}

type BsamTemplate struct {
	TemplateFormatVersion string        `yaml:"BCETemplateFormatVersion"`
	Transform             string        `yaml:"Transform"`
	Description           string        `yaml:"Description"`
	Resources             BsamResources `yaml:"Resources"`
}

type BsamResources map[string]*BsamFunctionResource

type BsamFunctionResource struct {
	Type       string                  `yaml:"Type"`
	Properties *BsamFunctionProperties `yaml:"Properties"`
}

type BsamFunctionProperties struct {
	CodeUri     string           `yaml:"CodeUri"`
	Handler     string           `yaml:"Handler"`
	Runtime     string           `yaml:"Runtime"`
	MemorySize  int              `yaml:"MemorySize"`
	Timeout     int              `yaml:"Timeout"`
	Environment *api.Environment `yaml:"Environment"`
}

func NewBsamTemplate() *BsamTemplate {
	return &BsamTemplate{
		TemplateFormatVersion: "2010-09-09",
		Transform:             "BCE::Serverless-2018-08-30",
		Description:           "",
		Resources:             NewBsamResources(),
	}
}

func NewBsamResources() BsamResources {
	return make(map[string]*BsamFunctionResource, 0)
}

func NewBsamFunctionResource(config *api.FunctionConfig) *BsamFunctionResource {
	return &BsamFunctionResource{
		Type:       "BCE::Serverless::Function",
		Properties: NewBsamFunctionProperties(config),
	}
}

func NewBsamFunctionProperties(config *api.FunctionConfig) *BsamFunctionProperties {
	return &BsamFunctionProperties{
		CodeUri:     filepath.Join(".", config.FunctionName),
		Handler:     config.Handler,
		Runtime:     config.Runtime,
		MemorySize:  *config.MemorySize,
		Timeout:     *config.Timeout,
		Environment: config.Environment,
	}
}
