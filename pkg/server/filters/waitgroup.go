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

	utilwaitgroup "github.com/baidu/openless/pkg/util/waitgroup"
)

// WithWaitGroup adds all non long-running requests to wait group, which is used for graceful shutdown.
func WithWaitGroup(handler http.Handler, wg *utilwaitgroup.SafeWaitGroup) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := wg.Add(1); err != nil {
			http.Error(w, "Apisever is shutting down.", http.StatusInternalServerError)
			return
		}
		defer wg.Done()

		handler.ServeHTTP(w, req)
	})
}
