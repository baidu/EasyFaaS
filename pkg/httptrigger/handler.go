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
	"net/http"

	"github.com/baidu/openless/pkg/api"

	routing "github.com/qiangxue/fasthttp-routing"
)

func HelloWorldHandler(c *routing.Context) error {
	c.SetStatusCode(http.StatusOK)
	c.WriteString("hello controller http trigger")
	return nil
}

func ProxyHandler(c *routing.Context) error {
	reqCtx := BuildContext(c)
	userID := c.Param("userID")
	if len(userID) == 0 {
		reqCtx.SetStatusCode(http.StatusBadRequest)
		reqCtx.WriteWithWarmLog(NewInvalidRequestException("missing userID", nil).Error())
		return nil
	}

	funcName := c.Param("functionName")
	if len(funcName) == 0 {
		c.SetStatusCode(http.StatusBadRequest)
		reqCtx.WriteWithWarmLog(NewInvalidRequestException("invalid function name", nil).Error())
		return nil
	}

	version := c.Param("version")
	if len(version) == 0 {
		version = "$LATEST"
	}
	authorization := string(c.Request.Header.Peek(api.HeaderAuthorization))

	ctx := ProxyContext{
		RequestID:     reqCtx.RequestID,
		Authorization: authorization,
		AccountID:     userID,
		FunctionName:  funcName,
		Version:       version,
		RouteCtx:      reqCtx,
		Logger:        reqCtx.Logger,
	}

	invokeType := string(c.Request.Header.Peek(api.HeaderInvokeType))
	if invokeType == "stream" {
		ctx.WithStreamMode = true
	}
	RequestController(&ctx)
	return nil
}

func InstallHandler() *routing.Router {
	router := routing.New()
	router.Get("/hello", HelloWorldHandler)
	router.Any(`/<userID:\w+>/<functionName>`, ProxyHandler)
	router.Any(`/<userID:\w+>/<functionName>/<version:\d*>`, ProxyHandler)
	return router
}
