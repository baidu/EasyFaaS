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
	"math/rand"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/baidu/openless/pkg/server/healthz"
	utilwaitgroup "github.com/baidu/openless/pkg/util/waitgroup"
)

// GenericServer contains state for a faas api server.
type GenericServer struct {
	// ShutdownTimeout is the timeout used for server shutdown. This specifies the timeout before server
	// gracefully shutdown returns.
	ShutdownTimeout time.Duration

	SecureServingInfo *SecureServingInfo

	// ExternalAddress is the address (hostname or IP and port) that should be used in
	// external (public internet) URLs for this GenericAPIServer.
	ExternalAddress string

	// Handler holds the handlers being used by this API server
	Handler *ServerHandler

	// PostStartHooks are each called after the server has started listening, in a separate go func for each
	// with no guarantee of ordering between them.  The map key is a name used for error reporting.
	// It may kill the process with a panic if it wishes to by returning an error.
	postStartHookLock    sync.Mutex
	postStartHooks       map[string]postStartHookEntry
	postStartHooksCalled bool

	preShutdownHookLock    sync.Mutex
	preShutdownHooks       map[string]preShutdownHookEntry
	preShutdownHooksCalled bool

	// healthz checks
	healthzLock    sync.Mutex
	healthzChecks  []healthz.HealthzChecker
	healthzCreated bool

	// HandlerChainWaitGroup allows you to wait for all chain handlers finish after the server shutdown.
	HandlerChainWaitGroup *utilwaitgroup.SafeWaitGroup
}

type preparedGenericServer struct {
	*GenericServer
}

// PrepareRun does post API installation setup steps.
func (s *GenericServer) PrepareRun() preparedGenericServer {
	runtime.GOMAXPROCS(runtime.NumCPU())
	s.installHealthz()
	rand.Seed(time.Now().UTC().UnixNano() + int64(os.Getpid()))

	return preparedGenericServer{s}
}

// Run spawns the secure http server. It only returns if stopCh is closed
// or the secure port cannot be listened on initially.
func (s preparedGenericServer) Run(stopCh <-chan struct{}) error {
	debug.SetGCPercent(1000)
	err := s.NonBlockingRun(stopCh)
	if err != nil {
		return err
	}

	<-stopCh

	err = s.RunPreShutdownHooks()
	if err != nil {
		return err
	}

	// Wait for all requests to finish, which are bounded by the RequestTimeout variable.
	s.HandlerChainWaitGroup.Wait()

	return nil
}

// NonBlockingRun spawns the http server. An error is
// returned if the port cannot be listened on.
func (s preparedGenericServer) NonBlockingRun(stopCh <-chan struct{}) error {
	// Use an internal stop channel to allow cleanup of the listeners on error.
	internalStopCh := make(chan struct{})

	s.ShutdownTimeout = 30 * time.Second

	if s.SecureServingInfo != nil && s.Handler != nil {
		if err := s.serveSecurely(internalStopCh); err != nil {
			close(internalStopCh)
			return err
		}
	}

	// Now that listener have bound successfully, it is the
	// responsibility of the caller to close the provided channel to
	// ensure cleanup.
	go func() {
		<-stopCh
		close(internalStopCh)
		s.HandlerChainWaitGroup.Wait()
	}()

	s.RunPostStartHooks(stopCh)

	return nil
}
