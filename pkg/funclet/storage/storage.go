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

// Package tmp
package storage

import (
	"fmt"
)

var storageMap = make(map[string]Storage, 0)

type Storage interface {
	Allocate(fpath string, size uint64) (storagePath string, err error)
	Recycle()
	PrepareXfs(path string) (err error)
	MountProjQuota(sourcePath string, targetPath string) (err error)
}

func RegisterStorage(name string, storage Storage) {
	storageMap[name] = storage
}

func GetStorage(name string) (storage Storage, err error) {
	storage, ok := storageMap[name]
	if !ok {
		return nil, fmt.Errorf("unknown tmp storage type %s", name)
	}
	return storage, nil
}
