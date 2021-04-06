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

import "time"

// ContentConfig contains settings that affect how objects are transformed when
// sent to the server.
type ContentConfig struct {
	// AcceptContentTypes specifies the types the client will accept and is optional.
	// If not set, ContentType will be used to define the Accept header
	AcceptContentTypes string
	// ContentType specifies the wire format used to communicate with the server.
	// This value will be set as the Accept header on requests made to the server, and
	// as the default content type on any object sent to the server. If not set,
	// "application/json" is used.
	ContentType string

	// BackendType xxx
	BackendType string

	// Connection specifies the Connection header
	Connection string

	// Connection specifies the KeepAlive header
	KeepAlive string

	ClientTimeout time.Duration
}

const (
	// defaultInternalAuthToken send internal auth token.
	// See baidu-faas-kun-486 (http://newicafe.baidu.com/issue/baidu-faas-kun-486/show)
	defaultInternalAuthToken = "cfc-auth-2018"
)

var (
	// DefaultInternalValidAuthTokens do internal auth verification
	DefaultInternalValidAuthTokens = []string{defaultInternalAuthToken}
)
