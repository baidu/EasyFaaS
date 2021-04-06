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

// Package noauth
package noauth

import (
	"testing"
)

func TestNewSigner(t *testing.T) {
	fc := noAuthFactory{}
	s, err := fc.NewSigner(nil)
	if err != nil {
		t.Errorf("new signer occurred error: %s", err)
		return
	}
	if s.Name() != authName {
		t.Errorf("signer expected %s, but got %s", authName, s.Name())
	}
	res := s.GetSignature(nil)
	if res != "" {
		t.Errorf("signer GetSignature expected empty, but got %s", res)
	}
}
