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
package tmp

import (
	"github.com/spf13/pflag"
	"github.com/baidu/openless/pkg/api"
)

type TmpStorageOption struct {
	TmpStorageType string
	TmpStoragePath string
	TmpStorageName string
	TmpStorageSize int64

	RunnerTmpSize string
}

func NewTmpStorageOption() *TmpStorageOption {
	return &TmpStorageOption{
		TmpStorageType: api.TmpStorageTypeLoop,
		TmpStoragePath: "/tmp/faas/",
		TmpStorageSize: 20 * 1024 * 1024 * 1024, // 20GB

		RunnerTmpSize: "500m",
	}
}
func (o *TmpStorageOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.TmpStorageType, "tmp-storage-type", o.TmpStorageType, "tmp storage type (eg: loop; disk)")
	fs.StringVar(&o.TmpStoragePath, "tmp-storage-path", o.TmpStoragePath, "tmp storage path")
	fs.Int64Var(&o.TmpStorageSize, "tmp-storage-size", o.TmpStorageSize, "tmp storage size")
}
