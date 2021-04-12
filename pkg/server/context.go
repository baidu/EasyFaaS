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

package server

import (
	"net/http"

	restful "github.com/emicklei/go-restful"

	"github.com/baidu/easyfaas/pkg/api"
	innerErr "github.com/baidu/easyfaas/pkg/error"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

type (
	// Context represents the context of the current HTTP request. It holds request and
	// response objects, path, path parameters, data and registered handler.
	Context struct {
		// Request() *http.Request
		// RestRequest() *restful.Request
		// Response() *restful.Response
		// Logger() ulog.LoggerContext
		// Context() context.Context
		// WithErrorLog(e error) innerErr.FinalError
		// WithWarnLog(e error) innerErr.FinalError
		requestID string
		request   *restful.Request
		response  *restful.Response
		logger    *logs.Logger
	}
)

// BuildContext xxx
func BuildContext(request *restful.Request, response *restful.Response) *Context {
	requestID := request.HeaderParameter(api.HeaderXRequestID)
	appname := request.HeaderParameter(api.AppNameKey)
	return &Context{
		requestID: requestID,
		request:   request,
		response:  response,
		logger:    logs.NewLogger().WithField("request_id", requestID).WithField(api.AppNameKey, appname),
	}
}

func (c *Context) RequestID() string {
	return c.requestID
}

func (c *Context) Request() *restful.Request {
	return c.request
}

func (c *Context) HTTPRequest() *http.Request {
	return c.request.Request
}

func (c *Context) Response() *restful.Response {
	return c.response
}

func (c *Context) Logger() *logs.Logger {
	return c.logger
}

func (c *Context) WithErrorLog(e error) innerErr.FinalError {
	c.Logger().AddCallerSkip(1).Error(e.Error())
	return innerErr.GenericKunFinalError(e)
}

func (c *Context) WithWarnLog(e error) innerErr.FinalError {
	c.Logger().AddCallerSkip(1).Warn(e.Error())
	return innerErr.GenericKunFinalError(e)
}

func WrapRestRouteFunc(h func(*Context)) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		c := BuildContext(request, response)
		h(c)
	}
}

func WrapRestFilterFunction(h func(c *Context, chain *restful.FilterChain)) restful.FilterFunction {
	return func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		c := BuildContext(request, response)
		h(c, chain)
	}
}
