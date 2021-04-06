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

// PatchType is the type of constants to support HTTP PATCH utilized by
// both the client and server that didn't make sense for a whole package to be
// dedicated to.
type PatchType string

const (
	// JSONPatchType defines a JSON document structure for expressing a sequence
	// of operations to apply to a JavaScript Object Notation (JSON) document.
	// See RFC 6902 (https://www.rfc-editor.org/rfc/rfc6902.txt) or
	// Jsonpatch (http://jsonpatch.com/) for more details.
	JSONPatchType PatchType = "application/json-patch+json"

	// MergePatchType defines the JSON merge patch format and processing rules.
	// See RFC 7386 (https://www.rfc-editor.org/rfc/rfc7386.txt)
	MergePatchType PatchType = "application/merge-patch+json"

	// StrategicMergePatchType xxx
	StrategicMergePatchType PatchType = "application/strategic-merge-patch+json"
)
