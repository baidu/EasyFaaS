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

package error

import (
	"github.com/baidu/easyfaas/pkg/util/json"
)

// AwsMessage
// Example:
// {
//    "message": "The security token included in the request is invalid."
// }
type AwsMessage struct {
	Message string `json:"message"`
}

// AwsErrorMessage
// Example:
// {
//    "errorMessage": "2018-05-10T06:42:59.753Z 5f46a493-541d-11e8-a58a-ed2562190b91 Task timed out after 3.00 seconds"
// }
type AwsErrorMessage struct {
	ErrorMessage string `json:"errorMessage"`
}

func (m *AwsErrorMessage) String() string {
	if m == nil {
		return ""
	}
	s, _ := json.Marshal(m)
	return string(s)
}
