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

package noauth

import (
	"github.com/baidu/easyfaas/pkg/auth"
	"github.com/baidu/easyfaas/pkg/auth/factory"
	"github.com/baidu/easyfaas/pkg/rest"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

const authName = "noauth"

type noauthSigner struct{}

type noAuthFactory struct{}

func init() {
	logs.Infof("register driver %s", authName)
	factory.Register(authName, &noAuthFactory{})
}

func (*noAuthFactory) NewSigner(parameters map[string]interface{}) (auth.Signer, error) {
	return NewSigner(), nil
}

func NewSigner() *noauthSigner {
	return &noauthSigner{}
}

func (s *noauthSigner) Name() string {
	return authName
}

func (s *noauthSigner) GetSignature(r *rest.Request) string {
	return ""
}
