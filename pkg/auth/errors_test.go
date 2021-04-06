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

// Package auth
package auth

import (
	"testing"
)

func TestError(t *testing.T) {
	ts := []struct {
		err error
		msg string
	}{
		{
			err: InitAuthSignerError{
				Name:    "noauth",
				Message: "test",
			},
			msg: "AuthSigner noauth init failed: test",
		},
		{
			err: ErrUnsupportedMethod{
				signerName: "noauth",
			},
			msg: "noauth: unsupported method",
		},
		{
			err: InvalidAuthSignerError{
				Name: "noauth",
			},
			msg: "AuthSigner not registered: noauth",
		},
	}
	for _, item := range ts {
		if item.msg != item.err.Error() {
			t.Errorf("err expected %s, but got %s", item.msg, item.err.Error())
		}
	}
}
