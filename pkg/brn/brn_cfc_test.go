// +build go1.7

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

package brn

import (
	"testing"
)

func TestDealFName(t *testing.T) {
	cases := []struct {
		input        string
		functionName string
		version      string
		alias        string
		uid          string
		err          error
	}{
		{
			input:        "brn:cloud:faas:bj:e1799855a16be6b26e2e0afad574dafe:function:function-name",
			functionName: "function-name",
			version:      "",
			alias:        "",
			uid:          "e1799855a16be6b26e2e0afad574dafe",
		},
		{
			input:        "brn:cloud:faas:bj:e1799855a16be6b26e2e0afad574dafe:function:function-name:$LATEST",
			functionName: "function-name",
			version:      "$LATEST",
			alias:        "",
			uid:          "e1799855a16be6b26e2e0afad574dafe",
		},
		{
			input:        "brn:cloud:faas:bj:e1799855a16be6b26e2e0afad574dafe:function:function-name:$LAEST",
			functionName: "",
			version:      "",
			alias:        "",
			uid:          "",
			err:          RegNotMatchErr,
		},
		{
			input:        "brn:cloud:faas:bj:e1799855a16be6b26e2e0afad574dafe:function:function-name:alias",
			functionName: "function-name",
			version:      "",
			alias:        "alias",
			uid:          "e1799855a16be6b26e2e0afad574dafe",
		},
		{
			input:        "e1799855a16be6b26e2e0afad574dafe:function-name",
			functionName: "function-name",
			version:      "",
			alias:        "",
			uid:          "e1799855a16be6b26e2e0afad574dafe",
		},
		{
			input:        "e1799855a16be6b26e2e0afad574dafe:function-name:$LATEST",
			functionName: "function-name",
			version:      "$LATEST",
			alias:        "",
			uid:          "e1799855a16be6b26e2e0afad574dafe",
		},
		{
			input:        "e1799855a16be6b26e2e0afad574dafe:----_function-name-----:$LATEST",
			functionName: "----_function-name-----",
			version:      "$LATEST",
			alias:        "",
			uid:          "e1799855a16be6b26e2e0afad574dafe",
		},
		{
			input:        "function-name:$LATEST",
			functionName: "function-name",
			version:      "$LATEST",
			alias:        "",
			uid:          "e1799855a16be6b26e2e0afad574dafe",
		},
		{
			input:        "function-name",
			functionName: "function-name",
			version:      "",
			alias:        "",
			uid:          "e1799855a16be6b26e2e0afad574dafe",
		},
		{
			input:        "function:oll:$LATEST",
			functionName: "",
			version:      "",
			alias:        "",
			uid:          "",
			err:          RegNotMatchErr,
		},
		{
			input:        "brn:cloud:faas:bj:function:function-name:$LATEST",
			functionName: "",
			version:      "",
			alias:        "",
			uid:          "",
			err:          RegNotMatchErr,
		},
		{
			input:        "brn:cloud:faas:bj:e1799855a16be6b26e2e0afad574dafe:function:woshifunction:3",
			functionName: "woshifunction",
			version:      "3",
			alias:        "",
			uid:          "e1799855a16be6b26e2e0afad574dafe",
		},
		{
			input:        "e1799855a16ce6b26e2e0afad574dafe:woshifunction:3",
			functionName: "",
			version:      "",
			alias:        "",
			uid:          "e1799855a16be6b26e2e0afad574dafe",
			err:          RegNotMatchErr,
		},
		{
			input:        "brn:cloud:faas:bj:e1799855a16be6b26e2e0afad574dafe:function:111:$LATEST",
			functionName: "111",
			version:      "$LATEST",
			alias:        "",
			uid:          "e1799855a16be6b26e2e0afad574dafe",
		},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			f, v, a, err := DealFName(tc.uid, tc.input)
			if tc.functionName != f {
				t.Errorf("Expected %q to parse as %v, but got %v", tc.input, tc.functionName, f)
			}
			if tc.version != v {
				t.Errorf("Expected %q to parse as %v, but got %v", tc.input, tc.version, v)
			}
			if tc.alias != a {
				t.Errorf("Expected %q to parse as %v, but got %v", tc.input, tc.alias, a)
			}

			if err == nil && tc.err != nil {
				t.Errorf("Expected err to be %v, but got nil", tc.err)
			} else if err != nil && tc.err == nil {
				t.Errorf("Expected err to be nil, but got %v", err)
			} else if err != nil && tc.err != nil && err.Error() != tc.err.Error() {
				t.Errorf("Expected err to be %v, but got %v", tc.err, err)
			}
		})
	}
}
