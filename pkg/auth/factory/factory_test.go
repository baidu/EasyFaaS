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

// Package factory
package factory

import (
	"testing"

	"github.com/baidu/easyfaas/pkg/auth"
	"github.com/baidu/easyfaas/pkg/rest"
)

func TestRegisterDuplicate(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("register expected error, but got nil")
		}
		t.Logf("registry err %s", r)
	}()
	Register("duplicate", &mockAuth{})
	Register("duplicate", &mockAuth{})
}

func TestEmptyAuthFactory(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("register expected error, but got nil")
		}
		t.Logf("registry err %s", r)
	}()
	Register("noauth", nil)
}

func TestNewSigner(t *testing.T) {
	s, err := NewSigner("mock-empty", nil)
	expectedErr := auth.InvalidAuthSignerError{"mock-empty"}
	if err != expectedErr {
		t.Errorf("init signer expected %s, but got:%s", expectedErr, err)
		return
	}
	Register("mock", &mockAuth{})
	s, err = NewSigner("mock", nil)
	if err != nil {
		t.Errorf("init signer failed:%s", err)
		return
	}
	if s.Name() != "mock" {
		t.Errorf("signer expected mock, but got %s", s.Name())
	}
}

type mockAuth struct{}

func (a *mockAuth) NewSigner(parameters map[string]interface{}) (auth.Signer, error) {
	return &mockAuth{}, nil
}
func (a *mockAuth) Name() string { return "mock" }

func (a *mockAuth) GetSignature(*rest.Request) string { return "" }
