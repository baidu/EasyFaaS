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

package options

import (
	"github.com/spf13/pflag"
	controllerClient "github.com/baidu/easyfaas/pkg/controller/client"
	genericoptions "github.com/baidu/easyfaas/pkg/server/options"
)

type HTTPTriggerOptions struct {
	RecommendedOptions      *genericoptions.RecommendedOptions
	ControllerClientOptions *controllerClient.ControllerClientOptions
}

func NewHTTPTriggerOptions() *HTTPTriggerOptions {
	return &HTTPTriggerOptions{
		RecommendedOptions:      genericoptions.NewRecommendedOptions(),
		ControllerClientOptions: controllerClient.NewControllerClientOptions(),
	}
}

func (o *HTTPTriggerOptions) AddFlags(fs *pflag.FlagSet) {
	o.RecommendedOptions.AddFlags(fs)
	o.ControllerClientOptions.AddFlags(fs)
}
