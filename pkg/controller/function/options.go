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

package function

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

type StorageCacheOptions struct {
	CacheExpiration time.Duration
}

func NewStorageCacheOptions() *StorageCacheOptions {
	return &StorageCacheOptions{
		CacheExpiration: 5 * time.Minute,
	}
}
func (s *StorageCacheOptions) AddFlags(prefix string, fs *pflag.FlagSet) {
	fs.DurationVar(&s.CacheExpiration, getFlagName(prefix, "cache-expiration"), s.CacheExpiration, "cache exipiration; unit seconds")
}

func getFlagName(prefix, flagName string) string {
	if prefix == "" {
		return flagName
	}
	return fmt.Sprintf("%s-%s", prefix, flagName)
}
