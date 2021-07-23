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

package httptrigger

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"io"
	"net/http"

	"github.com/baidu/easyfaas/cmd/httptrigger/options"
	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/brn"
	"github.com/baidu/easyfaas/pkg/controller/client"
	kunErr "github.com/baidu/easyfaas/pkg/error"
	"github.com/baidu/easyfaas/pkg/util/json"
)

func Init(runOptions *options.HTTPTriggerOptions) error {
	mc, err := client.NewControllerClient(runOptions.ControllerClientOptions)
	if err != nil {
		return err
	}
	NewProxy(mc)
	return nil
}

func InitWithClient(c client.ControllerClientInterface) {
	NewProxy(c)
	return
}

func RequestController(ctx *ProxyContext) {
	ir, err := buildRequest(ctx)
	if err != nil {
		ctx.RouteCtx.SetStatusCode(http.StatusBadRequest)
		ctx.RouteCtx.WriteWithErrorLog("invalid request body")
		return
	}
	cli := GetProxy().client
	resp, err := cli.Invoke(ir)
	if err != nil {
		ctx.RouteCtx.SetStatusCode(http.StatusBadGateway)
		ctx.RouteCtx.WriteWithErrorLog(NewBadGatewayException("function response error", err).Error())
		return
	}
	c, h, b, bs, e := makeResponse(resp, ctx)
	if e != nil {
		ctx.Logger.Errorf("make response error: %s", e)
		ctx.RouteCtx.SetStatusCode(http.StatusBadGateway)
		ctx.RouteCtx.WriteWithErrorLog(e.Error())
		return
	}
	for key, value := range h {
		ctx.RouteCtx.Response.Header.Set(key, value)
	}
	ctx.RouteCtx.SetStatusCode(c)
	if ctx.WithStreamMode {
		ctx.RouteCtx.Response.SetBodyStream(bs, -1)
	} else {
		ctx.RouteCtx.Write(b)
	}
	ctx.Logger.V(9).Info("invoke success")
	return
}

func makeResponse(resp *api.InvokeResponse, c *ProxyContext) (statusCode int, header map[string]string, body []byte, bodyStream io.ReadCloser, err error) {

	if resp.StatusCode() > 499 {
		statusCode = resp.StatusCode()
		err = NewInternalException(*resp.BodyString(), nil)
		c.Logger.Errorf("request controller error: %s", err)
		return
	}

	if resp.StatusCode() != http.StatusOK {
		statusCode = resp.StatusCode()
		header = *resp.Headers()
		body = resp.Body()
		return
	}

	if v, ok := resp.GetHeader(api.XBceFunctionError); ok && v == "Unhandled" {
		var errorMessage kunErr.AwsErrorMessage
		decoder := json.NewDecoder(bytes.NewReader(resp.Body()))
		if e := decoder.Decode(&errorMessage); e != nil {
			c.Logger.Errorf("decode response error: %s", e)
			err = e
			return
		}
		statusCode = http.StatusInternalServerError
		header = make(map[string]string)
		header["Content-Type"] = "application/json; charset=utf-8"
		body = []byte(errorMessage.String())
		c.Logger.Warn("invoke unhandled")
		return
	}

	if c.WithStreamMode {
		statusCode = http.StatusOK
		header = *resp.Headers()
		bodyStream = resp.BodyStream()
		return
	}

	decoder := json.NewDecoder(bytes.NewReader(resp.Body()))
	var pr api.ProxyResponse
	if e := decoder.Decode(&pr); e != nil {
		statusCode, header, body, err = simpleResponse(resp.Body(), *resp.Headers(), "text/plain")
		return
	}

	if pr.StatusCode < 100 || pr.StatusCode > 599 {
		statusCode, header, body, err = simpleResponse(resp.Body(), *resp.Headers(), "application/json")
		return
	}

	statusCode = pr.StatusCode
	header = pr.Headers
	if pr.IsBase64Encoded {
		decoded := make([]byte, 0)
		if _, e := base64.StdEncoding.Decode(decoded, []byte(pr.Body)); e != nil {
			statusCode, header, body, err = simpleResponse(resp.Body(), *resp.Headers(), "application/json")
			return
		}
		body = decoded
	} else {
		body = []byte(pr.Body)
	}
	return
}

func buildRequest(ctx *ProxyContext) (*api.InvokeRequest, error) {
	funcBrn := brn.GenerateFuncBrnString("bj", ctx.AccountID, ctx.FunctionName, ctx.Version)
	ctx.Logger.Infof("function brn is %s", funcBrn)
	ir := api.InvokeRequest{
		UserID:         ctx.AccountID,
		Authorization:  ctx.Authorization,
		FunctionBRN:    funcBrn,
		RequestID:      ctx.RequestID,
		WithBodyStream: ctx.WithStreamMode,
		LogType:        api.GetLogType(ctx.RouteCtx.Context),
		LogToBody:      api.GetLogToBody(ctx.RouteCtx.Context),
	}
	if ctx.WithStreamMode {
		bodyStream, err := requestBodyStream(ctx)
		if err != nil {
			return nil, err
		}
		ir.BodyStream = bodyStream
	} else {
		ir.Body = requestBodyString(ctx)
		ctx.Logger.Debugf("function request body is %s", *ir.Body)
	}
	return &ir, nil
}

func requestBodyStream(ctx *ProxyContext) (body *bufio.ReadWriter, err error) {
	read := bytes.NewBuffer(nil)
	write := bytes.NewBuffer(nil)
	buf := bufio.NewReadWriter(bufio.NewReader(read), bufio.NewWriter(write))
	err = ctx.RouteCtx.Request.BodyWriteTo(write)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func requestBodyString(ctx *ProxyContext) *string {
	bodyBytes, _ := json.Marshal(requestBody(ctx))
	bodyStr := string(bodyBytes)
	return &bodyStr
}

func requestBody(ctx *ProxyContext) *api.ProxyRequest {
	request := &ctx.RouteCtx.Request
	header := map[string]string{}
	request.Header.VisitAll(func(k, v []byte) {
		header[string(k)] = string(v)
	})

	query := map[string]string{}
	request.URI().QueryArgs().VisitAll(func(k, v []byte) {
		query[string(k)] = string(v)
	})

	req := api.ProxyRequest{
		HTTPMethod:            string(request.Header.Method()),
		Headers:               header,
		QueryStringParameters: query,
	}
	if v, ok := header["Content-Type"]; ok && v == "application/bson" {
		req.Body = base64.StdEncoding.EncodeToString(request.Body())
	} else {
		req.Body = string(request.Body())
	}
	return &req
}

func simpleResponse(rawByte []byte, rawHeader map[string]string, contentType string) (statusCode int, header map[string]string, body []byte, err error) {
	statusCode = http.StatusOK
	header = rawHeader
	header["Content-Type"] = contentType
	body = rawByte
	return
}
