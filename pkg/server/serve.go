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
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/baidu/openless/pkg/util/logs"
)

const (
	defaultKeepAlivePeriod = 3 * time.Minute
)

// serveSecurely runs the secure http server. It fails only if certificates cannot
// be loaded or the initial listen call fails. The actual server loop (stoppable by closing
// stopCh) runs in a go routine, i.e. serveSecurely does not block.
func (s *GenericServer) serveSecurely(stopCh <-chan struct{}) error {
	if s.SecureServingInfo.Listener == nil {
		return fmt.Errorf("listener must not be nil")
	}
	secureServer := &http.Server{
		Addr:           s.SecureServingInfo.Listener.Addr().String(),
		Handler:        s.Handler,
		MaxHeaderBytes: 1 << 20,
	}
	err := RunServer(secureServer, s.SecureServingInfo.Listener, s.ShutdownTimeout, stopCh)
	return err
}

// RunServer listens on the given port if listener is not given,
// then spawns a go-routine continuously serving
// until the stopCh is closed. This function does not block.
func RunServer(server *http.Server, ln net.Listener, shutDownTimeout time.Duration, stopCh <-chan struct{}) error {

	// Shutdown server gracefully.
	go func() {
		<-stopCh
		ctx, cancel := context.WithTimeout(context.Background(), shutDownTimeout)
		server.Shutdown(ctx)
		cancel()
	}()

	go func() {
		var listener net.Listener
		switch ln.(type) {
		case *net.TCPListener:
			listener = tcpKeepAliveListener{ln.(*net.TCPListener)}
		default:
			listener = ln
		}
		err := server.Serve(listener)

		msg := "Stopped listening"
		select {
		case <-stopCh:
			logs.V(6).Info(msg)
		default:
			panic(fmt.Sprintf("%s due to error: %v", msg, err))
		}
	}()

	return nil
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
//
// Copied from Go 1.7.2 net/http/server.go
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(defaultKeepAlivePeriod)
	return tc, nil
}
