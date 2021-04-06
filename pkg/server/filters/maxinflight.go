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

package filters

import (
	"net/http"
)

// Constant for the retry-after interval on rate limiting.
// TODO: maybe make this dynamic? or user-adjustable?
const retryAfter = "1"

// WithMaxInFlightLimit limits the number of in-flight requests to buffer size of the passed in channel.
func WithMaxInFlightLimit(handler http.Handler, limit int) http.Handler {
	if limit == 0 {
		return handler
	}
	var ch chan bool
	if limit != 0 {
		ch = make(chan bool, limit)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case ch <- true:
			defer func() { <-ch }()
			handler.ServeHTTP(w, r)
		default:
			tooManyRequests(r, w)
		}
	})
}

func tooManyRequests(req *http.Request, w http.ResponseWriter) {
	// Return a 503 status indicating "Too Many Requests" of server overloading
	// instead of 429, because 429 means user throttle limit exceed
	w.Header().Set("Retry-After", retryAfter)
	http.Error(w, "Too many requests, please try again later.", http.StatusServiceUnavailable)
}
