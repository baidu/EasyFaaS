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
	"time"

	"github.com/baidu/openless/pkg/util/cache"
)

type CacheType string

const (
	CacheTypeFunction CacheType = "function"
	CacheTypeAlias    CacheType = "alias"
	CacheTypeRuntime  CacheType = "runtime"
)

const (
	defaultCacheExpiration      = 10 * time.Minute
	defaultCacheCleanupInterval = 30 * time.Minute
)

func CacheKey(cacheType CacheType, key string) string {
	return string(cacheType) + ":" + key
}

type StorageCache struct {
	cacheExpirationConfigs map[CacheType]time.Duration
	*cache.Cache
}

func NewStorageCache(expirationConfig map[CacheType]time.Duration) *StorageCache {
	return &StorageCache{
		expirationConfig,
		cache.New(defaultCacheExpiration, defaultCacheCleanupInterval),
	}
}

func DefaultCacheExpirationConfigs() map[CacheType]time.Duration {
	conf := make(map[CacheType]time.Duration, 0)
	conf[CacheTypeFunction] = defaultCacheExpiration
	conf[CacheTypeAlias] = defaultCacheExpiration
	conf[CacheTypeRuntime] = defaultCacheExpiration
	return conf
}

func (s *StorageCache) CacheExpiration(cacheType CacheType) time.Duration {
	if time, ok := s.cacheExpirationConfigs[cacheType]; ok {
		return time
	}
	return defaultCacheExpiration
}
