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

package bos

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/baidu/easyfaas/pkg/repository"
	"github.com/baidu/easyfaas/pkg/repository/factory"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

const (
	driverName      = "bos"
	DefaultBasePath = "/var/faas/tmp"
)

type DriverParameters struct {
	BasePath string
}

func init() {
	logs.Infof("register driver %s", driverName)
	factory.Register(driverName, &bosDriverFactory{})
}

type bosDriverFactory struct{}

func (factory *bosDriverFactory) Create(parameters map[string]interface{}) (repository.StorageDriver, error) {
	return FromParameters(parameters)
}

type driver struct {
	basePath string
}

func FromParameters(parameters map[string]interface{}) (*driver, error) {
	params, err := fromParametersImpl(parameters)
	if err != nil || params == nil {
		return nil, err
	}
	return New(*params)
}

func fromParametersImpl(parameters map[string]interface{}) (*DriverParameters, error) {
	var basePath string
	val, ok := parameters["basePath"]
	if val != nil {
		if item, ok := val.(string); ok {
			basePath = item
		}
	}
	if basePath == "" || !ok {
		basePath = DefaultBasePath
	}
	params := &DriverParameters{
		BasePath: basePath,
	}
	return params, nil
}

func New(params DriverParameters) (*driver, error) {
	return &driver{
		basePath: params.BasePath,
	}, nil
}

func (d *driver) Name() string {
	return driverName
}
func (d *driver) Fetch(codeLocation string) (path string, err error) {
	logger := logs.NewLogger()
	defer logger.TimeTrack(time.Now(), fmt.Sprintf("Download code location codeLocation %s", codeLocation))
	codeLocationURL, err := url.Parse(codeLocation)
	if err != nil {
		return "", repository.InvalidPathError{Path: codeLocation, DriverName: driverName}
	}
	filename := filepath.Join(d.basePath, codeLocationURL.EscapedPath())

	logger.Infof("Mkdir [%s] Download tmp file to [%s]", filepath.Dir(filename), filename)

	if err := os.MkdirAll(filepath.Dir(filename), 0777); err != nil {
		return "", err
	}

	logger.Infof("Start http request [%s]", codeLocation)
	resp, err := http.Get(codeLocation)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	logger.Infof("Start write to file [%s]", filename)
	f, err := os.Create(filename)
	if err != nil {
		os.Remove(filename)
		return "", err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(filename)
		return "", err
	}

	return filename, nil
}
