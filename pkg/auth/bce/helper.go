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

package bce

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

func GetCanonicalTime(now time.Time) string {
	year, mon, day := now.UTC().Date()
	hour, min, sec := now.UTC().Clock()
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02dZ", year, mon, day, hour, min, sec)
}

func HmacSha256Hex(key, message string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func GetNormalizedString(s string, skipSlash bool) string {
	r := url.QueryEscape(s)
	// https://github.com/aws/aws-sdk-go/blob/60293cacbc1ed27eb78374e1b24a3ea55a74d1b6/aws/signer/v4/v4.go#L635
	r = strings.Replace(r, "+", "%20", -1)
	if skipSlash {
		r = strings.Replace(r, "%2F", "/", -1)
	}
	return r
}

func GetCanonicalQueryString(query url.Values) string {
	result := []string{}
	for k := range query {
		if k == "authorization" {
			continue
		}
		v := query.Get(k)
		if len(v) == 0 {
			result = append(result, GetNormalizedString(k, false)+"=")
		} else {
			result = append(result, GetNormalizedString(k, false)+"="+GetNormalizedString(v, false))
		}
	}
	sort.Strings(result)
	return strings.Join(result, "&")
}

func GetCanonicalHeaders(headers http.Header, r rule) (string, []string) {
	standardHeaders := []string{
		"host", "content-md5", "content-length", "content-type",
	}
	var result []string
	var signHeaders []string
	for key := range headers {
		keyLower := strings.ToLower(key)
		value := headers.Get(key)
		if r.IsValid(keyLower) && (strings.HasPrefix(keyLower, "x-cloud-") || contains(standardHeaders, keyLower)) {
			result = append(result, keyLower+":"+GetNormalizedString(value, false))
			signHeaders = append(signHeaders, keyLower)
		}
	}
	sort.Strings(result)
	sort.Strings(signHeaders)
	return strings.Join(result, "\n"), signHeaders
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
