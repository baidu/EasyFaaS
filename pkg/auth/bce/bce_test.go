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

// Package cloud
package bce

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/baidu/openless/pkg/rest"

	"github.com/baidu/openless/pkg/auth"
)

func TestBceAuthFactory_NewSigner(t *testing.T) {
	fc := bceAuthFactory{}
	ts := []struct {
		params map[string]interface{}
		err    error
	}{
		{
			params: map[string]interface{}{
				"sk": "xxxx",
			},
			err: auth.InitAuthSignerError{
				Name:    authName,
				Message: fmt.Sprintf("no %s parameter provided", paramAccessKey),
			},
		},
		{
			params: map[string]interface{}{
				"ak": "xxxx",
			},
			err: auth.InitAuthSignerError{
				Name:    authName,
				Message: fmt.Sprintf("no %s parameter provided", paramSecretKey),
			},
		},
		{
			params: map[string]interface{}{
				"ak": "xxxx",
				"sk": "xxxx",
			},
			err: nil,
		},
	}
	for _, item := range ts {
		s, err := fc.NewSigner(item.params)
		if err != item.err {
			t.Errorf("new signer expected %s, but got %s", item.err, err)
		}
		if item.err == nil {
			t.Logf("new signer %s", s.Name())
		}
	}
}

func TestBceSigner_GetSignature(t *testing.T) {
	s := getSigner()
	baseURL, _ := url.Parse("http://127.0.0.1/v1/functions/test")
	req := rest.NewRequest(nil, "GET", baseURL, "v1", rest.ContentConfig{}, nil, 0)
	req.SetHeader("Content-Length", "13256")
	req.SetHeader("Content-MD5", "ujOLK9GE1xdbYdfKvfI1BA==")
	req.SetHeader("Host", "bos.qasandbox.bcetest.baidu.com")
	req.SetHeader("x-cloud-content-sha256", "4604e6530e5a06dec3099e77977cab6c7c88eae21005f6edd66deffc203a5e6e")
	req.SetHeader("x-cloud-date", "2016-01-04T06:12:04Z")
	req.SetHeader("x-cloud-request-id", "f98023ac-1189-412b-93f5-859970585dce")

	req.Param("foo", "bar")
	req.Param("path", "/aaa/bbb/ccc")
	res := s.GetSignature(req)
	t.Logf("get signature %s", res)
}

func getSigner() auth.Signer {
	fc := bceAuthFactory{}
	params := map[string]interface{}{
		"ak": "xxxx",
		"sk": "xxxx",
	}
	signer, _ := fc.NewSigner(params)
	return signer
}
