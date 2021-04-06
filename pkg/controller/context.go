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

package controller

import (
	"github.com/baidu/openless/cmd/controller/options"
	"github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/brn"
	"github.com/baidu/openless/pkg/controller/rtctrl"
	"github.com/baidu/openless/pkg/util/logs"
)

type InvokeContext struct {
	RunOptions *options.ControllerOptions

	ExternalRequestID string // RequestID from external request
	RequestID         string // System inner unique RequestID

	AccountID     string    // user account id from external request
	CallerUser    *api.User // credential user info of function's caller
	OwnerUser     *api.User // credential user info of function's owner
	Authorization string

	Function *api.GetFunctionOutput
	Runtime  *api.RuntimeConfiguration

	Input     *rtctrl.InvocationInput
	Output    *rtctrl.InvocationOutput
	Statistic *rtctrl.InvocationStatistic

	// controller invoke request
	Request *api.InvokeProxyRequest
	// controller invoke response
	Response *api.InvokeProxyResponse

	Logger  *logs.Logger
	Clients *Clients

	FunctionName string
	FunctionBRN  string
	Qualifier    string
	Brn          brn.BRN

	LogType   api.LogType
	LogToBody bool

	Metrics        *InvokeMetrics
	WithStreamMode bool

	InvokeType  string
	TriggerType string
}
