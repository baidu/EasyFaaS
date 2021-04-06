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

// Package rtctrl
package rtctrl

import (
	"fmt"
	"strings"
)

type RuntimeStateUnmatched struct {
	RuntimeID     string
	CurrentState  RuntimeStateType
	ExpectedState []RuntimeStateType
}

func (e RuntimeStateUnmatched) Error() string {
	return fmt.Sprintf("runtime %s state did not match op, current state %s, expected state %s", e.RuntimeID, e.CurrentState, strings.Join(e.ExpectedState, ","))
}

// RuntimeMatchError
type RuntimeMatchError struct {
	Reason string
}

func (e RuntimeMatchError) Error() string {
	return fmt.Sprintf("runtime is not matched: %s", e.Reason)
}

// RuntimeReleaseError: release runtime failed
type RuntimeReleaseError struct {
	RuntimeID string
	Reason    string
}

func (e RuntimeReleaseError) Error() string {
	return fmt.Sprintf("runtime %s is not matched: %s", e.RuntimeID, e.Reason)
}

// RuntimeReleaseError: release runtime failed
type RuntimeSyncError struct {
	RuntimeID string
	Reason    string
}

func (e RuntimeSyncError) Error() string {
	return fmt.Sprintf("runtime %s is not matched: %s", e.RuntimeID, e.Reason)
}

// RuntimeNotExist: runtime does not exist
type RuntimeNotExist struct {
	RuntimeID string
}

func (e RuntimeNotExist) Error() string {
	return fmt.Sprintf("runtime %s does not exist", e.RuntimeID)
}

// RuntimeInfoError
type RuntimeInfoError struct {
	RuntimeID string
}

func (e RuntimeInfoError) Error() string {
	return fmt.Sprintf("information of runtime %s is invaild", e.RuntimeID)
}

// RuntimeNoNeedToReset
type RuntimeNoNeedToReset struct {
	RuntimeID string
}

func (e RuntimeNoNeedToReset) Error() string {
	return fmt.Sprintf("runtime is no need to reset: %s", e.RuntimeID)
}
