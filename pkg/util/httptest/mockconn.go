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
	"bytes"
	"io"
	"net"
	"sync"
	"time"
)

type mockConn struct {
	conn           net.Conn
	closed         bool
	received       *bytes.Buffer
	responseReader io.Reader
	mx             *sync.RWMutex
}

func newMockConn(received *bytes.Buffer, responseReader io.Reader) (m *mockConn) {
	var mx sync.RWMutex
	m = &mockConn{
		received:       received,
		responseReader: responseReader,
		mx:             &mx,
	}
	return
}

func (m *mockConn) Read(p []byte) (n int, e error) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	n, e = m.responseReader.Read(p)
	return
}

func (m *mockConn) Write(p []byte) (n int, e error) {
	m.mx.Lock()
	defer m.mx.Unlock()
	n, e = m.received.Write(p)
	return
}
func (c *mockConn) Close() error {
	c.closed = true
	return nil
}

func (c *mockConn) Closed() bool {
	return c.closed
}

func (c *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 1234,
		Zone: "",
	}
}

func (c *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 4321,
		Zone: "",
	}
}

func (c *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}
