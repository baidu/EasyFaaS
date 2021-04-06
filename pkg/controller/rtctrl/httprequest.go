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

package rtctrl

import (
	"context"
	"github.com/baidu/openless/pkg/api"
	"mime"
	"net/http"
)

func ConvertProxyRequestToHTTP(reqInfo *RequestInfo) (request *http.Request, cancel func()) {
	req := reqInfo.Input.Request
	ctx, cancel := context.WithCancel(context.Background())
	request, _ = http.NewRequestWithContext(ctx, "POST", "http://unix", reqInfo.Input.Request.BodyStream)
	for k, v := range req.Headers {
		request.Header.Set(k, v)
	}
	// TODO: add client context
	request.Header.Set(api.HeaderXRequestID, reqInfo.RequestID)
	return request, cancel
}

func ConvertHTTPResponseToProxy(rsp *http.Response, reqInfo *RequestInfo) {
	output := reqInfo.Output.Output
	response := output.Response
	response.StatusCode = rsp.StatusCode
	if response.StatusCode == http.StatusOK {
		// in stream mode, we would not transfer body data by funcResult
		reqInfo.InvokeResult(StatusSuccess, "")
	} else {
		// in stream mode, we would not transfer body data by funcResult
		reqInfo.InvokeResult(StatusFailed, "")
	}
	response.Headers = make(map[string]string)

	for k, v := range rsp.Header {
		if len(v) > 0 {
			response.Headers[k] = v[0]
		}
	}

	contentType := rsp.Header.Get("Content-Type")
	if contentType == "" {
		response.IsBase64Encoded = true
	} else {
		mediaType, _, _ := mime.ParseMediaType(contentType)
		switch mediaType {
		case "application/json", "text/xml", "text/html":
			response.IsBase64Encoded = false
		default:
			response.IsBase64Encoded = true
		}
	}
	response.BodyStream = rsp.Body
	return
}
