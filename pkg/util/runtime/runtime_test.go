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

package runtime

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestHandleCrash(t *testing.T) {
	defer func() {
		if x := recover(); x == nil {
			t.Errorf("Expected a panic to recover from")
		}
	}()
	defer HandleCrash()
	panic("Test Panic")
}

func TestCustomHandleCrash(t *testing.T) {
	old := PanicHandlers
	defer func() { PanicHandlers = old }()
	var result interface{}
	PanicHandlers = []func(interface{}){
		func(r interface{}) {
			result = r
		},
	}
	func() {
		defer func() {
			if x := recover(); x == nil {
				t.Errorf("Expected a panic to recover from")
			}
		}()
		defer HandleCrash()
		panic("test")
	}()
	if result != "test" {
		t.Errorf("did not receive custom handler")
	}
}

func TestCustomHandleError(t *testing.T) {
	old := ErrorHandlers
	defer func() { ErrorHandlers = old }()
	var result error
	ErrorHandlers = []func(error){
		func(err error) {
			result = err
		},
	}
	err := fmt.Errorf("test")
	HandleError(err)
	if result != err {
		t.Errorf("did not receive custom handler")
	}
}

func TestGetCaller(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// TODO: Add test cases.
		{
			name: "test",
			want: "testing.tRunner",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCaller(); got != tt.want {
				t.Errorf("GetCaller() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecoverFromPanic(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "panic",
			args: args{
				err: errors.New("for test"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testPanic(); err != nil {
				if strings.Contains(fmt.Sprintf("%v", err), "error: recovered from panic \"Hi\"") {
					t.Fatalf("error not match: %s\n", err)
				}
			}
		})
	}
}

func testPanic() (err error) {
	defer RecoverFromPanic(&err)
	panic("Hi")
}
