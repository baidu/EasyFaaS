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

package cache

import (
	"errors"
	"time"
)

var (
	ErrorCacheNotAvilable = errors.New("watch cache not available")
)

type WatchCacheStore struct {
	cache     *Cache
	available bool
}

func NewWatchCacheStore(defaultExpiration, cleanupInterval time.Duration) *WatchCacheStore {
	cache := New(defaultExpiration, cleanupInterval)
	wc := &WatchCacheStore{
		cache:     cache,
		available: true,
	}
	return wc
}

func (wc *WatchCacheStore) Flush() {
	wc.cache.Flush()
}

func (wc *WatchCacheStore) SetAvailable(available bool) {
	if available == wc.available {
		return
	}
	if available == false {
		wc.Flush()
	}
	wc.available = available
}

func (wc *WatchCacheStore) Get(k string) (interface{}, error) {
	if !wc.available {
		return nil, ErrorCacheNotAvilable
	}
	value, ok := wc.cache.Get(k)
	if !ok {
		return nil, errors.New("watch cache not have this key")
	}
	return value, nil
}

func (wc *WatchCacheStore) Set(k string, x interface{}, d time.Duration) error {
	if !wc.available {
		return ErrorCacheNotAvilable
	}
	wc.cache.Set(k, x, d)
	return nil
}

func (wc *WatchCacheStore) SetDefault(k string, x interface{}) error {
	return wc.Set(k, x, DefaultExpiration)
}

func (wc *WatchCacheStore) Delete(k string) {
	wc.cache.Delete(k)
}

func (wc *WatchCacheStore) OnEvicted(f func(string, interface{})) {
	wc.cache.OnEvicted(f)
}
