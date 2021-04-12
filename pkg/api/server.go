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

package api

import (
	"github.com/qiangxue/fasthttp-routing"
)

const (
	HeaderXRequestID    = "X-easyfaas-Request-Id"
	HeaderXAccountID    = "X-easyfaas-Account-Id"
	HeaderAuthorization = "Authorization"
	AppNameKey          = "app"
	HeaderInvokeType    = "X-easyfaas-Invoke-Type"
	HeaderLogType       = "Log-Type"
	HeaderLogToBody     = "Log-To-Body"
	HeaderXAuthToken    = "X-Auth-Token"

	BceFaasUIDKey          = "BCE-FAAS-UID"
	BceFaasTriggerKey      = "X-easyfaas-Faas-Trigger"
	XBceFunctionError      = "X-easyfaas-Function-Error"
	HeadereasyfaasExecTime = "X-easyfaas-Function-Exectime"
	HeaderLogResult        = "X-Bce-Log-Result"

	QueryLogType   = "logType"
	QueryLogToBody = "logToBody"
)

func GetLogType(c *routing.Context) LogType {
	var t LogType
	if t = LogType(c.Request.Header.Peek(HeaderLogType)); t.Valid() {
		return t
	}
	if t = LogType(c.Request.URI().QueryArgs().Peek(QueryLogType)); t.Valid() {
		return t
	}
	return LogTypeNone
}

func GetLogToBody(c *routing.Context) bool {
	if b := string(c.Request.Header.Peek(HeaderLogToBody)); b == "true" {
		return true
	}
	if b := string(c.Request.URI().QueryArgs().Peek(QueryLogToBody)); b == "true" {
		return true
	}
	return false
}
