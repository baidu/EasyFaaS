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

// Package rtctrl
package rtctrl

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
)

func TestListenerFromAddress(t *testing.T) {
	// unix
	unixAddr := fmt.Sprintf("unix:///tmp/%s.sock", uuid.New().String())
	ln, err := ListenerFromAddress(unixAddr, os.ModePerm)
	if err != nil {
		t.Errorf("listen from address %s failed: %s", unixAddr, err)
		return
	}
	ln.Close()

	// tcp
	tcpAddr := "tcp://127.0.0.1:33333"
	ln, err = ListenerFromAddress(tcpAddr, os.ModePerm)
	if err != nil {
		t.Errorf("listen from address %s failed: %s", tcpAddr, err)
		return
	}
	ln.Close()

	// tcp with query
	tcpAddr1 := "tcp://127.0.0.1:33334?interface=lo0&maskbits=8"
	ln, err = ListenerFromAddress(tcpAddr1, os.ModePerm)
	if err == nil {
		ln.Close()
	}
}

func TestFilterNetAddr(t *testing.T) {
	filterNetAddr("lo0", "127.0.0.1", 8)
}
