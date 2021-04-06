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

package rest

import (
	"net/url"
	"time"
)

// Set of resp. Codes that we backoff for.
// In general these should be errors that indicate a server is overloaded.
// These shouldn't be configured by any user, we set them based on conventions
// described in
// var serverIsOverloadedSet = sets.NewInt(429)
// var maxResponseCode = 499

type BackoffManager interface {
	UpdateBackoff(url *url.URL, err error, responseCode int)
	CalculateBackoff(url *url.URL) time.Duration
	Sleep(d time.Duration)
}

// NoBackoff is a stub implementation, can be used for mocking or else as a default.
type NoBackoff struct {
}

func (n *NoBackoff) UpdateBackoff(url *url.URL, err error, responseCode int) {
	// do nothing.
}

func (n *NoBackoff) CalculateBackoff(url *url.URL) time.Duration {
	return 0 * time.Second
}

func (n *NoBackoff) Sleep(d time.Duration) {
	time.Sleep(d)
}
