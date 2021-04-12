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

package client

import (
	"github.com/baidu/easyfaas/cmd/controller/options"
	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/controller"
	"github.com/baidu/easyfaas/pkg/funclet/client"
	"github.com/baidu/easyfaas/pkg/util/id"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

type ControllerCallClient struct {
	controllerClient controller.ControllerInterface
	funcletClient    client.FuncletInterface
	runOptions       *options.ControllerOptions
}

func NewControllerCallClient(controllerClient controller.ControllerInterface,
	funcletClient client.FuncletInterface,
	opt *options.ControllerOptions) *ControllerCallClient {
	return &ControllerCallClient{controllerClient, funcletClient, opt}
}

func (mc *ControllerCallClient) Invoke(ir *api.InvokeRequest) (response *api.InvokeResponse, err error) {
	h := make(map[string]string, 0)
	reqID := id.GetRequestID()

	clientMode := controller.ClientModeHTTPTrigger
	ctx := controller.InvokeContext{
		RunOptions:        mc.runOptions,
		ExternalRequestID: ir.RequestID,
		RequestID:         reqID,
		AccountID:         ir.UserID,
		Authorization:     ir.Authorization,
		Clients:           mc.controllerClient.NewClients(clientMode),
		Response:          api.NewInvokeProxyResponseWithRequestID(ir.RequestID),
		Logger: logs.NewLogger().WithField("request_id", reqID).
			WithField("external_request_id", ir.RequestID),
		FunctionBRN:    ir.FunctionBRN,
		Qualifier:      ir.Qualifier,
		WithStreamMode: ir.WithBodyStream,
		InvokeType:     api.InvokeTypeHttpTrigger,
		TriggerType:    api.TriggerTypeHTTP,
	}

	if ir.WithBodyStream {
		ctx.Request = api.NewInvokeProxyRequest(h, nil, ir.BodyStream)
	} else {
		ctx.Request = api.NewInvokeProxyRequest(h, []byte(*ir.Body), nil)
	}

	if mc.runOptions.RecommendedOptions.Features.EnableMetrics {
		ctx.Metrics = controller.NewInvokeMetrics(ir.RequestID)
	}

	mc.controllerClient.Do(&ctx)

	response = api.NewInvokeResponse()
	response.SetStatusCode(ctx.Response.StatusCode)
	response.SetHeaders(&ctx.Response.Headers)
	if ctx.WithStreamMode {
		response.SetBodyStream(ctx.Response.BodyStream)
	} else {
		response.SetBody(ctx.Response.Body)
	}

	if ctx.Metrics != nil {
		ctx.Metrics.Overall()
		ctx.Metrics.WriteSummary(mc.runOptions.RecommendedOptions.Features.SummaryOverheadMs)
	}

	return response, nil
}
