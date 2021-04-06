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

// Package httptest
package httptest

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
)

// A test response implementing the http.Hijacker interface.
type ResponseHijacker struct {
	httptest.ResponseRecorder
	in          *bufio.Reader
	out         *bufio.Writer
	HijackError error
	HijackConn  *mockConn
}

func NewResponseHijacker(hijackedInputData []byte) *ResponseHijacker {
	rh := &ResponseHijacker{
		ResponseRecorder: *httptest.NewRecorder(),
		in:               bufio.NewReader(bytes.NewBuffer(hijackedInputData)),
	}
	rh.HijackConn = newMockConn(rh.ResponseRecorder.Body, rh.in)
	rh.out = bufio.NewWriter(rh.ResponseRecorder.Body)
	return rh
}

func (r *ResponseHijacker) Header() http.Header {
	return r.ResponseRecorder.Header()
}

func (r *ResponseHijacker) WriteHeader(stateCode int) {
	r.ResponseRecorder.WriteHeader(stateCode)
}

func (h *ResponseHijacker) Write(buf []byte) (int, error) {
	return h.ResponseRecorder.Write(buf)
}

func (h *ResponseHijacker) WriteString(str string) (int, error) {
	return h.ResponseRecorder.WriteString(str)
}

func (h *ResponseHijacker) Flush() {
	h.ResponseRecorder.Flush()
}

func (h *ResponseHijacker) Result() *http.Response {
	return h.ResponseRecorder.Result()
}

func (h *ResponseHijacker) Body() *bytes.Buffer {
	return h.ResponseRecorder.Body
}

func (h *ResponseHijacker) Code() int {
	return h.ResponseRecorder.Code
}

func (h *ResponseHijacker) Flushed() bool {
	return h.ResponseRecorder.Flushed
}

func (r *ResponseHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.HijackConn,
		bufio.NewReadWriter(bufio.NewReader(r.HijackConn.responseReader), bufio.NewWriter(r.HijackConn)),
		r.HijackError
}
