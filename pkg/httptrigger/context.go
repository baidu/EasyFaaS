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
	routing "github.com/qiangxue/fasthttp-routing"
	"github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/util/id"
	"github.com/baidu/openless/pkg/util/logs"
)

func BuildContext(c *routing.Context) *Context {
	requestID := string(c.Request.Header.Peek(api.HeaderXRequestID))
	if len(requestID) == 0 {
		requestID = id.GetRequestID()
	}
	return &Context{
		RequestID: requestID,
		Logger:    logs.NewLogger().WithField("request_id", requestID),
		Context:   c,
	}
}

func (c *Context) WriteWithErrorLog(msg string) {
	c.Logger.Error(msg)
	c.WriteString(msg)
}

func (c *Context) WriteWithWarmLog(msg string) {
	c.Logger.Warn(msg)
	c.WriteString(msg)
}
