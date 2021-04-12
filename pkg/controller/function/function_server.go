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

package function

import (
	"github.com/baidu/easyfaas/pkg/controller/registry"

	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

// DataStorer xxx
type DataStorer interface {
	GetFunction(input *api.GetFunctionInput) (*api.GetFunctionOutput, bool, error)
	GetAlias(input *api.GetAliasInput) (*api.GetAliasOutput, bool, error)
	GetRuntimeConfiguration(input *api.GetRuntimeConfigurationInput) (*api.RuntimeConfiguration, bool, error)
}

// functionServerClient is used to get function and policy meta from apiserver
type functionServerClient struct {
	rpcClient registry.Registry
	cache     *StorageCache
}

// NewDataStorer create a apiserver client
func NewDataStorer(o *registry.Options, storageCache *StorageCache) (ds DataStorer, err error) {
	rpcClient, err := registry.NewRegistry(o)
	if err != nil {
		return nil, err
	}
	ds = &functionServerClient{
		rpcClient: rpcClient,
		cache:     storageCache,
	}
	return
}

func (f *functionServerClient) GetFunction(input *api.GetFunctionInput) (output *api.GetFunctionOutput, hitCache bool, err error) {
	if input.WithCache {
		v, ok := f.cache.Get(CacheKey(CacheTypeFunction, *input.FunctionName))
		if ok {
			output = v.(*api.GetFunctionOutput)
			hitCache = true
			logs.Debugf("get function cache %s", *input.FunctionName)
			return
		}
	}
	output, err = f.rpcClient.GetFunction(input)
	if err != nil {
		return
	}

	if input.WithCache {
		f.cache.Set(CacheKey(CacheTypeFunction, *input.FunctionName), output, f.cache.CacheExpiration(CacheTypeFunction))
	}
	return
}

func (f *functionServerClient) GetAlias(input *api.GetAliasInput) (output *api.GetAliasOutput, hitCache bool, err error) {
	if input.WithCache && input.FunctionBrn != "" {
		v, ok := f.cache.Get(CacheKey(CacheTypeAlias, input.FunctionBrn))
		if ok {
			output = v.(*api.GetAliasOutput)
			hitCache = true
			logs.Debugf("get function cache %s", input.FunctionBrn)
			return
		}
	}
	output, err = f.rpcClient.GetAlias(input)
	if err != nil {
		return
	}

	if input.WithCache && input.FunctionBrn != "" {
		f.cache.Set(CacheKey(CacheTypeAlias, input.FunctionBrn), output, f.cache.CacheExpiration(CacheTypeAlias))
	}
	return
}

func (f *functionServerClient) GetRuntimeConfiguration(input *api.GetRuntimeConfigurationInput) (conf *api.RuntimeConfiguration, hitCache bool, err error) {
	v, ok := f.cache.Get(CacheKey(CacheTypeRuntime, input.RuntimeName))
	if ok {
		conf = v.(*api.RuntimeConfiguration)
		hitCache = true
		logs.Debugf("get runtime cache %s", input.RuntimeName)
		return
	}

	conf, err = f.rpcClient.GetRuntimeConfiguration(input)
	if err != nil {
		return
	}

	f.cache.Set(CacheKey(CacheTypeRuntime, input.RuntimeName), conf, f.cache.CacheExpiration(CacheTypeRuntime))
	return
}
