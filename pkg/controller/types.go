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

// Package controller
package controller

import (
	"github.com/baidu/easyfaas/cmd/controller/options"
	"github.com/baidu/easyfaas/pkg/controller/function"
	"github.com/baidu/easyfaas/pkg/controller/rtctrl"
	"github.com/baidu/easyfaas/pkg/funclet/client"
)

type Controller struct {
	runOptions            *options.ControllerOptions
	FuncletClient         client.FuncletInterface
	runtimeDispatcher     rtctrl.RuntimeDispatcher
	runtimeControl        rtctrl.Control
	dataStorer            function.DataStorer
	insideDataStorer      function.DataStorer
	httpTriggerDataStorer function.DataStorer
}

// Clients save all clients to make rpc calls
type Clients struct {
	DataStorer     function.DataStorer
	RuntimeControl rtctrl.Control
	FuncletClient  client.FuncletInterface
}

func (controller *Controller) NewClients(clientMode string) *Clients {
	switch clientMode {
	case ClientModeInside:
		return &Clients{
			DataStorer:     controller.insideDataStorer,
			RuntimeControl: controller.runtimeControl,
			FuncletClient:  controller.FuncletClient,
		}
	case ClientModeHTTPTrigger:
		return &Clients{
			DataStorer:     controller.httpTriggerDataStorer,
			RuntimeControl: controller.runtimeControl,
			FuncletClient:  controller.FuncletClient,
		}
	default:
		return &Clients{
			DataStorer:     controller.dataStorer,
			RuntimeControl: controller.runtimeControl,
			FuncletClient:  controller.FuncletClient,
		}
	}
}

type logToBodyResult struct {
	FuncError string `json:"FunctionError,omitempty"`
	LogResult string `json:"LogResult,omitempty"`
	Payload   string `json:"Payload"`
}

const (
	ClientModeCommon      = "common"
	ClientModeInside      = "inside"
	ClientModeHTTPTrigger = "httpTrigger"
)
