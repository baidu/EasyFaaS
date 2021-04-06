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

package registry

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	innerErr "github.com/baidu/openless/pkg/error"

	"github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/auth"
	_ "github.com/baidu/openless/pkg/auth/bce"
	"github.com/baidu/openless/pkg/auth/factory"
	_ "github.com/baidu/openless/pkg/auth/noauth"
	"github.com/baidu/openless/pkg/rest"
	"github.com/baidu/openless/pkg/util/json"
)

type Registry interface {
	GetFunction(input *api.GetFunctionInput) (*api.GetFunctionOutput, error)
	GetAlias(input *api.GetAliasInput) (*api.GetAliasOutput, error)
	GetRuntimeConfiguration(input *api.GetRuntimeConfigurationInput) (*api.RuntimeConfiguration, error)
}

func NewRegistry(o *Options) (r Registry, err error) {
	o.Auth.Params = make(map[string]interface{})
	if len(o.Auth.ParamStr) != 0 {
		if err := json.Unmarshal([]byte(o.Auth.ParamStr), &o.Auth.Params); err != nil {
			return nil, err
		}
	}
	signer, err := factory.NewSigner(o.Auth.Name, o.Auth.Params)
	if err != nil {
		return nil, err
	}
	baseURL, _ := url.Parse("http://" + strings.Replace(o.Endpoint, "http://", "", 1))
	conf := rest.ContentConfig{
		ClientTimeout: 30 * time.Second,
	}
	restClient, _ := rest.NewRESTClient(baseURL, o.Version, conf, nil)
	rc := &RegistryClient{
		authTransfer: o.AuthTransfer,
		client:       restClient,
		signer:       signer,
	}
	return rc, nil
}

type RegistryClient struct {
	authTransfer bool
	client       *rest.RESTClient
	signer       auth.Signer
}

func (c *RegistryClient) GetFunction(input *api.GetFunctionInput) (*api.GetFunctionOutput, error) {
	req := c.client.Get().
		Resource("functions/" + *input.FunctionName)
	req.SetHeader("Host", req.URL().Host)
	req.SetHeader(api.HeaderXRequestID, input.RequestID)

	if input.SimpleAuth {
		req.SetHeader(api.HeaderXAuthToken, "cfc-auth-2018")
		req.SetHeader(api.BceFaasUIDKey, "root")
	}
	if c.authTransfer && input.Authorization != "" {
		req.SetHeader(api.HeaderAuthorization, input.Authorization)
	} else {
		req.SetHeader(api.HeaderXAccountID, input.AccountID)
		req.SetHeader(api.HeaderAuthorization, c.signer.GetSignature(req))
	}

	var out api.GetFunctionOutput
	result := req.Do()
	err := result.Into(&out)
	if err != nil && result.GetStatusCode() == http.StatusNotFound {
		finalErr := innerErr.NewResourceNotFoundException("get function from data storer failed", err)
		return &out, finalErr
	}
	return &out, err
}

func (c *RegistryClient) GetAlias(input *api.GetAliasInput) (*api.GetAliasOutput, error) {
	// TODO: not a common api
	req := c.client.Get().
		Resource("functions/aliases/" + input.FunctionBrn)
	req.SetHeader("Host", req.URL().Host)
	req.SetHeader(api.HeaderXRequestID, input.RequestID)

	if input.SimpleAuth {
		req.SetHeader(api.HeaderXAuthToken, "cfc-auth-2018")
		req.SetHeader(api.BceFaasUIDKey, "root")
	}
	if c.authTransfer && input.Authorization != "" {
		req.SetHeader(api.HeaderAuthorization, input.Authorization)
	} else {
		req.SetHeader(api.HeaderXAccountID, input.AccountID)
		req.SetHeader(api.HeaderAuthorization, c.signer.GetSignature(req))
	}

	var out api.GetAliasOutput
	result := req.Do()
	err := result.Into(&out)
	if err != nil && result.GetStatusCode() == http.StatusNotFound {
		finalErr := innerErr.NewResourceNotFoundException("get alias from data storer failed", err)
		return &out, finalErr
	}
	return &out, err
}

func (c *RegistryClient) GetRuntimeConfiguration(input *api.GetRuntimeConfigurationInput) (*api.RuntimeConfiguration, error) {
	req := c.client.Get().
		Resource("runtimes/" + input.RuntimeName + "/configuration")
	req.SetHeader("Host", req.URL().Host)
	req.SetHeader(api.HeaderXRequestID, input.RequestID)

	if c.authTransfer && input.Authorization != "" {
		req.SetHeader(api.HeaderAuthorization, input.Authorization)
	} else {
		req.SetHeader(api.HeaderAuthorization, c.signer.GetSignature(req))
	}

	var out api.RuntimeConfiguration
	result := req.Do()
	err := result.Into(&out)
	if err != nil && result.GetStatusCode() == http.StatusNotFound {
		finalErr := innerErr.NewResourceNotFoundException("get runtime configuration from data storer failed", err)
		return &out, finalErr

	}
	return &out, err
}
