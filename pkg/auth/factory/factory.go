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

package factory

import (
	"fmt"

	"github.com/baidu/openless/pkg/auth"
)

var authFactories = make(map[string]AuthFactory)

type AuthFactory interface {
	NewSigner(parameters map[string]interface{}) (auth.Signer, error)
}

func Register(name string, factory AuthFactory) {
	if factory == nil {
		panic("Must not provide nil AuthFactory")
	}
	_, registered := authFactories[name]
	if registered {
		panic(fmt.Sprintf("AuthFactory named %s already registered", name))
	}

	authFactories[name] = factory
}

func NewSigner(name string, parameters map[string]interface{}) (auth.Signer, error) {
	authFactory, ok := authFactories[name]
	if !ok {
		return nil, auth.InvalidAuthSignerError{name}
	}
	return authFactory.NewSigner(parameters)
}
