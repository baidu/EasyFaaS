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

package filesystem

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/baidu/easyfaas/pkg/repository"
	"github.com/baidu/easyfaas/pkg/repository/factory"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

const (
	driverName      = "filesystem"
	DefaultBasePath = "/var/faas/tmp"
	DefaultFileType = "zip"
)

type DriverParameters struct {
	BasePath string
}

func init() {
	logs.Infof("register driver %s", driverName)
	factory.Register(driverName, &filesystemDriverFactory{})
}

type filesystemDriverFactory struct{}

func (factory *filesystemDriverFactory) Create(parameters map[string]interface{}) (repository.StorageDriver, error) {
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
func (d *driver) Fetch(codeLocation string) (filePath string, err error) {
	logger := logs.NewLogger()
	defer logger.TimeTrack(time.Now(), fmt.Sprintf("Download code location codeLocation %s", codeLocation))
	if _, err := os.Stat(codeLocation); os.IsNotExist(err) {
		return "", err
	}
	// TODO: support more file type
	sourceFile, err := os.Open(codeLocation)
	if err != nil {
		return "", err
	}
	defer sourceFile.Close()
	head := make([]byte, 4)
	sourceFile.Read(head)
	if !isZip(head) {
		logs.Errorf("zip file header is %s", string(head))
		return "", fmt.Errorf("%s is not a zip file", codeLocation)
	}

	filename := filepath.Join(d.basePath, path.Base(codeLocation))

	logger.Infof("Mkdir [%s] Download tmp file to [%s]", filepath.Dir(filename), filename)

	if err := os.MkdirAll(filepath.Dir(filename), 0777); err != nil {
		return "", err
	}

	logger.Infof("Start write to file [%s]", filename)
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		os.Remove(filename)
		return "", err
	}
	defer f.Close()
	sourceFile.Seek(0, io.SeekStart)
	if _, err := io.Copy(f, sourceFile); err != nil {
		os.Remove(filename)
		return "", err
	}
	return filename, nil
}

func isZip(buf []byte) bool {
	return len(buf) > 3 &&
		buf[0] == 0x50 && buf[1] == 0x4B &&
		(buf[2] == 0x3 || buf[2] == 0x5 || buf[2] == 0x7) &&
		(buf[3] == 0x4 || buf[3] == 0x6 || buf[3] == 0x8)
}
