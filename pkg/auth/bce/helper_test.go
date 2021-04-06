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

// Package cloud
package bce

import (
	"net/url"
	"testing"
)

var accessKey = "b5e478e040214973a2c44d49fba0adb4"
var secretKey = "49589fb2c3da4041b8fd9cd9bfbdeef3"

func TestHmacSha256Hex(t *testing.T) {
	authStringPrefix := "cloud-auth-v1/" + accessKey + "/2016-01-04T06:12:04Z/1800"
	result := HmacSha256Hex(secretKey, authStringPrefix)
	expectResult := "86518e34da86dc9b477e5542301beea75a5fec9fde17920ddfa15b026e220a3b"
	if result != expectResult {
		t.Errorf("want '%s', get '%s'", expectResult, result)
	}
}

func TestGetCanonicalQueryString(t *testing.T) {
	params := url.Values{}
	params.Add("foo", "bar")
	params.Add("path", "/aaa/bbb/ccc")
	res := GetCanonicalQueryString(params)
	expectResult := "foo=bar&path=%2Faaa%2Fbbb%2Fccc"
	if res != expectResult {
		t.Errorf("want '%s', get '%s'", expectResult, res)
	}
}
