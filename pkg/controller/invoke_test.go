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

// Package controller
package controller

import (
	"strings"
	"testing"
)

func TestParseBrn(t *testing.T) {
	ctx := &InvokeContext{
		FunctionBRN: "brn:cloud:faas:bj:cd64f99c69d7c404b61de0a4f1865834:function:hello-tmp:$LATEST",
	}
	brn, err := parseBrn(ctx)
	t.Logf("brn resource %s", brn.Resource)
	if strings.HasSuffix(brn.Resource, "$LATEST") {
		t.Logf("resource is latest")
	}
	t.Logf("err %s", err)
}
