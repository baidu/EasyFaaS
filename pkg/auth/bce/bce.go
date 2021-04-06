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

package bce

import (
	"fmt"
	"strings"
	"time"

	"github.com/baidu/openless/pkg/auth"
	"github.com/baidu/openless/pkg/auth/factory"
	"github.com/baidu/openless/pkg/rest"
	"github.com/baidu/openless/pkg/util/logs"
)

const authName = "cloud"

const (
	paramAccessKey = "ak"
	paramSecretKey = "sk"
)

const (
	defaultExpire           = 3000
	defaultWithSignedHeader = true
)

type bceSigner struct {
	accessKey        string
	secretKey        string
	expire           int64
	ignoredHeaders   rules // ignored Headers
	withSignedHeader bool
}

type bceAuthFactory struct{}

func init() {
	logs.Infof("register driver %s", authName)
	factory.Register(authName, &bceAuthFactory{})
}

func (*bceAuthFactory) NewSigner(parameters map[string]interface{}) (auth.Signer, error) {
	return FromParameters(parameters)
}

func FromParameters(parameters map[string]interface{}) (*bceSigner, error) {
	ak, ok := parameters[paramAccessKey]
	if !ok || fmt.Sprint(ak) == "" {
		return nil, auth.InitAuthSignerError{
			Name:    authName,
			Message: fmt.Sprintf("no %s parameter provided", paramAccessKey),
		}
	}
	sk, ok := parameters[paramSecretKey]
	if !ok || fmt.Sprint(ak) == "" {
		return nil, auth.InitAuthSignerError{
			Name:    authName,
			Message: fmt.Sprintf("no %s parameter provided", paramSecretKey),
		}
	}

	return NewSigner(fmt.Sprint(ak), fmt.Sprint(sk)), nil
}

func NewSigner(ak, sk string) *bceSigner {
	return &bceSigner{
		accessKey: ak,
		secretKey: sk,
		expire:    defaultExpire,
		ignoredHeaders: rules{
			blacklist{
				mapRule{
					"authorization": struct{}{},
					"user-agent":    struct{}{},
					"cloud-faas-uid":  struct{}{},
					"x-auth-token":  struct{}{},
					"app":           struct{}{},
				},
			},
		},
		withSignedHeader: defaultWithSignedHeader,
	}
}

func (s *bceSigner) Name() string {
	return authName
}

func (s *bceSigner) GetSignature(r *rest.Request) string {
	stringPrefix := fmt.Sprintf("cloud-auth-v1/%s/%s/%d", s.accessKey, GetCanonicalTime(time.Now()), defaultExpire)
	signKey := HmacSha256Hex(s.secretKey, stringPrefix)

	canonicalURI := GetNormalizedString(r.URL().Path, true)
	canonicalQueryString := GetCanonicalQueryString(r.GetParams())
	canonicalHeaders, signHeaders := GetCanonicalHeaders(r.Header(), s.ignoredHeaders)

	canonicalRequest := r.Verb() + "\n" + canonicalURI + "\n" + canonicalQueryString + "\n" + canonicalHeaders
	signature := HmacSha256Hex(signKey, canonicalRequest)

	var result string
	if s.withSignedHeader == true {
		result = fmt.Sprintf("%s/%s/%s", stringPrefix, strings.Join(signHeaders, ";"), signature)
	} else {
		result = fmt.Sprintf("%s//%s", stringPrefix, signature)
	}
	return result
}
