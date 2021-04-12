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
	"bufio"
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	innerErr "github.com/baidu/easyfaas/pkg/error"

	routing "github.com/qiangxue/fasthttp-routing"

	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/util/id"
	"github.com/baidu/easyfaas/pkg/util/json"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

func (controller *Controller) HealthzHandler(c *routing.Context) error {
	c.SetStatusCode(http.StatusOK)
	c.WriteString("hello controller")
	return nil
}

func (controller *Controller) InvokeHandler(c *routing.Context) error {
	externalRequestID := string(c.Request.Header.Peek(api.HeaderXRequestID))
	invokeType := strings.ToLower(string(c.Request.Header.Peek(api.HeaderInvokeType)))
	triggerType := strings.ToLower(string(c.Request.Header.Peek(api.BceFaasTriggerKey)))
	accountID := string(c.Request.Header.Peek(api.HeaderXAccountID))
	authorization := string(c.Request.Header.Peek(api.HeaderAuthorization))
	requestID := id.GetRequestID()
	if externalRequestID == "" {
		externalRequestID = requestID
	}

	if invokeType == "" {
		invokeType = api.InvokeTypeCommon
	}
	if triggerType == "" {
		triggerType = api.TriggerTypeGeneric
	}

	var clientMode string
	if invokeType == api.InvokeTypeEvent || invokeType == api.InvokeTypeMqhub {
		clientMode = ClientModeInside
	}
	if triggerType == api.TriggerTypeVscode {
		clientMode = ClientModeInside
	}

	ctx := InvokeContext{
		RunOptions:        controller.runOptions,
		ExternalRequestID: externalRequestID,
		RequestID:         requestID,
		AccountID:         accountID,
		Authorization:     authorization,
		CallerUser: &api.User{
			ID: accountID,
		},
		Clients:  controller.NewClients(clientMode),
		Request:  generateInvokeRequest(c),
		Response: api.NewInvokeProxyResponseWithRequestID(externalRequestID),
		Logger: logs.NewLogger().WithField("request_id", requestID).
			WithField("external_request_id", externalRequestID).WithField("invoke-type", invokeType),
		LogType:     api.GetLogType(c),
		LogToBody:   api.GetLogToBody(c),
		InvokeType:  invokeType,
		TriggerType: triggerType,
	}

	funcName := c.Param("functionName")
	if !api.RegfunctionName.MatchString(funcName) {
		ctx.FunctionBRN = funcName
	} else {
		qualifier := string(c.QueryArgs().Peek("Qualifier"))
		if qualifier == "" {
			c.Response.SetStatusCode(http.StatusBadRequest)
			err := innerErr.NewInvalidParameterValueException("missing query parameter Qualifier", nil)
			bodyData, _ := json.Marshal(err)
			c.Response.SetBody(bodyData)
			return err
		}
		ctx.FunctionName = funcName
		ctx.Qualifier = qualifier
	}

	// TODO: just for test;remove it after apiserver deployed in production
	// http stream mode should read from the function runtime configuration
	if invokeType == api.InvokeTypeStream {
		ctx.WithStreamMode = true
	}

	if controller.runOptions.RecommendedOptions.Features.EnableMetrics {
		ctx.Metrics = NewInvokeMetrics(requestID)
	}

	startTime := time.Now()
	defer ctx.Logger.TimeTrack(startTime, "Invocation Total time")

	controller.Invoke(c, ctx)

	return nil
}

func (controller *Controller) Invoke(c *routing.Context, ctx InvokeContext) {
	if ctx.InvokeType == api.InvokeTypeEvent {
		go controller.Do(&ctx)
		c.SetStatusCode(http.StatusCreated)
	} else {
		controller.Do(&ctx)
		makeHTTPResponse(c, &ctx)
	}
}

func (controller *Controller) ListRuntimesHandler(c *routing.Context) error {
	runtimes := controller.runtimeDispatcher.RuntimeList()
	body, err := json.Marshal(runtimes)
	if err != nil {
		return err
	}
	c.Response.SetBody(body)
	return nil
}

func (controller *Controller) GetResourceHandler(c *routing.Context) error {
	resource := controller.runtimeDispatcher.ResourceStatistics()
	body, err := json.Marshal(resource)
	if err != nil {
		return err
	}
	c.Response.SetBody(body)
	return nil
}

func (controller *Controller) InvalidateRuntime(c *routing.Context) error {
	runtimeID := c.Param("runtimeID")
	runtime, err := controller.runtimeDispatcher.GetRuntime(runtimeID)
	if err != nil {
		c.Response.SetStatusCode(http.StatusNotFound)
		c.Response.AppendBodyString("can not find runtime " + runtimeID)
	}

	if runtime == nil {
		c.Response.SetStatusCode(http.StatusNotFound)
		c.Response.AppendBodyString("can not find runtime " + runtimeID)
		return nil
	}
	logs.Infof("invalidate runtime %d manually", runtimeID)
	runtime.Invalidate()
	return nil
}

func generateInvokeRequest(c *routing.Context) *api.InvokeProxyRequest {
	hMap := make(map[string]string)
	c.Request.Header.VisitAll(func(key, value []byte) {
		hMap[string(key)] = string(value)
	})
	body, err := generateBodyStream(c)
	if err != nil {
		logs.Errorf("generate body stream failed: %s", err)
		// TODO: error？
		return api.NewInvokeProxyRequest(hMap, nil, nil)
	}
	invokeType := strings.ToLower(string(c.Request.Header.Peek(api.HeaderInvokeType)))
	if invokeType == api.InvokeTypeEvent {
		bodyBytes, _ := ioutil.ReadAll(body)
		return api.NewInvokeProxyRequest(hMap, bodyBytes, nil)
	}
	return api.NewInvokeProxyRequest(hMap, nil, body)
}

func generateBodyStream(c *routing.Context) (body *bufio.ReadWriter, err error) {
	bodyBuffer := bytes.NewBuffer(nil)
	buf := bufio.NewReadWriter(bufio.NewReader(bodyBuffer), bufio.NewWriter(bodyBuffer))
	err = c.Request.BodyWriteTo(bodyBuffer)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func makeHTTPResponse(c *routing.Context, ctx *InvokeContext) {
	c.SetStatusCode(ctx.Response.StatusCode)
	for k, v := range ctx.Response.Headers {
		c.Response.Header.Set(k, v)
	}
	if ctx.Response.BodyStream != nil {
		c.Response.SetBodyStream(ctx.Response.BodyStream, -1)
		return
	}
	c.Write(ctx.Response.Body)
}
