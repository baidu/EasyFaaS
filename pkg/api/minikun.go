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
	"io"
	"strings"
)

const MinMemorySize int64 = 128

type InvokeRequest struct {
	UserID         string
	Authorization  string
	FunctionName   string
	Qualifier      string
	FunctionBRN    string
	Body           *string
	BodyStream     io.ReadWriter
	WithBodyStream bool
	LogToBody      bool
	LogType        LogType
	RequestID      string
}

type InvokeResponse struct {
	IsBase64Encoded bool
	code            int
	headers         map[string]string
	bodyRaw         []byte
	bodyStream      io.ReadCloser
}

func NewInvokeResponse() *InvokeResponse {
	h := make(map[string]string, 0)
	b := make([]byte, 0)
	return &InvokeResponse{
		headers: h,
		bodyRaw: b,
	}
}

func (iresp *InvokeResponse) Body() []byte {
	return iresp.bodyRaw
}

func (iresp *InvokeResponse) BodyStream() io.ReadCloser {
	return iresp.bodyStream
}

func (iresp *InvokeResponse) BodyString() *string {
	s := string(iresp.bodyRaw)
	return &s
}

func (iresp *InvokeResponse) SetBody(b []byte) {
	iresp.bodyRaw = b
}

func (iresp *InvokeResponse) SetBodyStream(bs io.ReadCloser) {
	iresp.bodyStream = bs
}

func (iresp *InvokeResponse) Headers() *map[string]string {
	return &iresp.headers
}

func (iresp *InvokeResponse) GetHeader(k string) (v string, ok bool) {
	v, ok = iresp.headers[k]
	return
}

func (iresp *InvokeResponse) SetHeaders(hMap *map[string]string) {
	iresp.headers = *hMap
}

func (iresp *InvokeResponse) SetHeader(k, v string) {
	iresp.headers[k] = v
}

func (iresp *InvokeResponse) StatusCode() int {
	return iresp.code
}

func (iresp *InvokeResponse) SetStatusCode(c int) {
	iresp.code = c
}

type InvokeProxyRequest struct {
	Headers         map[string]string
	Body            []byte
	BodyStream      io.ReadWriter
	IsBase64Encoded bool
}

type InvokeProxyResponse struct {
	StatusCode      int
	Headers         map[string]string
	Body            []byte
	BodyStream      io.ReadCloser
	IsBase64Encoded bool
}

func NewInvokeProxyRequest(headers map[string]string, body []byte, bodyStream io.ReadWriter) *InvokeProxyRequest {
	return &InvokeProxyRequest{
		Headers:    headers,
		Body:       body,
		BodyStream: bodyStream,
	}
}

func (ir *InvokeProxyRequest) SetHeader(k string, v string) {
	hMap := ir.Headers
	hMap[k] = v
}

func (ir *InvokeProxyRequest) SetBody(msg []byte) {
	ir.Body = msg
}

func NewInvokeProxyResponse() *InvokeProxyResponse {
	hMap := make(map[string]string, 0)
	return &InvokeProxyResponse{
		Headers: hMap,
	}
}

func NewInvokeProxyResponseWithRequestID(requestID string) *InvokeProxyResponse {
	hMap := make(map[string]string, 0)
	hMap[HeaderXRequestID] = requestID
	return &InvokeProxyResponse{
		Headers: hMap,
	}
}
func (ir *InvokeProxyResponse) SetStatusCode(c int) {
	ir.StatusCode = c
}

func (ir *InvokeProxyResponse) SetHeader(k string, v string) {
	hMap := ir.Headers
	hMap[k] = v
}

func (ir *InvokeProxyResponse) SetBody(msg []byte) {
	ir.Body = msg
}

type LogType string

const (
	LogTypeNone = "None"
	LogTypeTail = "Tail"
)

// Valid check if logtype is valid
func (t LogType) Valid() bool {
	if strings.EqualFold(string(t), LogTypeNone) ||
		strings.EqualFold(string(t), LogTypeTail) {
		return true
	}
	return false
}

// IsLogTypeTail check if logtype is Tail
func (t LogType) IsLogTypeTail() bool {
	return strings.EqualFold(string(t), LogTypeTail)
}

type ServiceResource struct {
	Capacity    *Resource
	Allocatable *Resource
	Used        *Resource
	Marked      *Resource
	Default     *Resource
	BaseMemory  uint64
}

func (r *ServiceResource) Copy() *ServiceResource {
	rp := ServiceResource{
		Capacity:    r.Capacity.Copy(),
		Allocatable: r.Allocatable.Copy(),
		Used:        r.Used.Copy(),
		Marked:      r.Marked.Copy(),
		Default:     r.Default.Copy(),
		BaseMemory:  r.BaseMemory,
	}
	return &rp
}

type ScaleUpRecommendation struct {
	TargetMemory     string
	TargetContainer  string
	MergedContainers []string
}

type ScaleDownRecommendation struct {
	TargetContainer string
	ResetContainers []string
}

type InvokeType = string

const (
	InvokeTypeCommon      InvokeType = "common"
	InvokeTypeEvent       InvokeType = "event"
	InvokeTypeStream      InvokeType = "stream"
	InvokeTypeHttpTrigger InvokeType = "httpTrigger"
	InvokeTypeMqhub       InvokeType = "mqhub"
)

type TriggerType = string

const (
	TriggerTypeHTTP    TriggerType = "faas-http-trigger"
	TriggerTypeCrontab TriggerType = "faas-crontab-trigger"
	TriggerTypeKafka   TriggerType = "kafka"
	TriggerTypeVscode  TriggerType = "vscode"
	TriggerTypeGeneric TriggerType = "generic"
)
