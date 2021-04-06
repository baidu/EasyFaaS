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
	"net"
	"net/http"
	"runtime"
	"time"

	genericfilters "github.com/baidu/openless/pkg/server/filters"
	"github.com/baidu/openless/pkg/server/healthz"
	"github.com/baidu/openless/pkg/server/routes"
	"github.com/baidu/openless/pkg/util/waitgroup"
	"github.com/baidu/openless/pkg/version"
)

// Config is a structure used to configure a GenericAPIServer.
// Its members are sorted roughly in order of importance for composers.
type Config struct {
	// SecureServingInfo is required to serve https
	SecureServingInfo *SecureServingInfo

	EnableProfiling bool
	// Requires generic profiling enabled
	EnableContentionProfiling bool
	EnableMetrics             bool
	SummaryOverheadMs         int

	// Version will enable the /version endpoint if non-nil
	Version *version.Version

	// ExternalAddress is the host name to use for external (public internet) facing URLs (e.g. Swagger)
	// Will default to a value based on secure serving info and available ipv4 IPs.
	ExternalAddress string

	// HandlerChainWaitGroup allows you to wait for all chain handlers exit after the server shutdown.
	HandlerChainWaitGroup *waitgroup.SafeWaitGroup

	// The default set of healthz checks. There might be more added via AddHealthzChecks dynamically.
	HealthzChecks []healthz.HealthzChecker

	// ===========================================================================
	// Fields you probably don't care about changing
	// ===========================================================================

	// BuildHandlerChainFunc allows you to build custom handler chains by decorating the apiHandler.
	BuildHandlerChainFunc func(apiHandler http.Handler, c *Config) http.Handler

	// If specified, all requests except those which match the LongRunningFunc predicate will timeout
	// after this duration.
	RequestTimeout time.Duration

	// MaxRequestsInFlight is the maximum number of parallel non-long-running requests. Every further
	// request has to wait. Applies only to non-mutating requests.
	MaxRequestsInFlight int
}

type RecommendedConfig struct {
	Config
}

// BuildHandlerChainFunc is a type for functions that build handler chain
type BuildHandlerChainFunc func(apiHandler http.Handler, c *Config) http.Handler

// SecureServingInfo xxx
type SecureServingInfo struct {
	// Listener is the secure server network listener.
	Listener net.Listener
}

// NewConfig returns a Config struct with the default values
func NewConfig() *Config {
	return &Config{
		HandlerChainWaitGroup: new(waitgroup.SafeWaitGroup),
		HealthzChecks:         []healthz.HealthzChecker{healthz.PingHealthz},
		BuildHandlerChainFunc: DefaultBuildHandlerChain,
		Version:               version.Get(),
		MaxRequestsInFlight:   4000,
		RequestTimeout:        time.Duration(340) * time.Second,

		EnableProfiling:           false,
		EnableContentionProfiling: false,
		EnableMetrics:             true,
		SummaryOverheadMs:         1,
	}
}

// NewRecommendedConfig returns a RecommendedConfig struct with the default values
func NewRecommendedConfig() *RecommendedConfig {
	return &RecommendedConfig{
		Config: *NewConfig(),
	}
}

// CompletedConfig xxx
type CompletedConfig struct {
	*Config
}

// Complete fills in any fields not set that are required to have valid data and can be derived
// from other fields. If you're going to `ApplyOptions`, do that first. It's mutating the receiver.
func (c *Config) Complete() CompletedConfig {
	return CompletedConfig{c}
}

// New creates a new server which logically combines the handling chain with the passed server.
// name is used to differentiate for logging. The handler chain in particular can be difficult as it starts delgating.
func (c CompletedConfig) New(name string) (*GenericServer, error) {
	handlerChainBuilder := func(handler http.Handler) http.Handler {
		return c.BuildHandlerChainFunc(handler, c.Config)
	}
	handler := NewServerHandler(name, handlerChainBuilder, nil)
	s := &GenericServer{
		HandlerChainWaitGroup: c.HandlerChainWaitGroup,

		SecureServingInfo: c.SecureServingInfo,
		ExternalAddress:   c.ExternalAddress,
		Handler:           handler,

		postStartHooks:   map[string]postStartHookEntry{},
		preShutdownHooks: map[string]preShutdownHookEntry{},

		healthzChecks: c.HealthzChecks,
	}

	installAPI(s, c.Config)

	return s, nil
}

// DefaultBuildHandlerChain xxx
func DefaultBuildHandlerChain(apiHandler http.Handler, c *Config) http.Handler {
	handler := genericfilters.WithRequestID(apiHandler)
	handler = genericfilters.WithMaxInFlightLimit(handler, c.MaxRequestsInFlight)
	handler = genericfilters.WithWaitGroup(handler, c.HandlerChainWaitGroup)
	handler = genericfilters.WithTimeoutFilter(handler, c.RequestTimeout)
	return handler
}

func installAPI(s *GenericServer, c *Config) {
	if c.EnableProfiling {
		routes.Profiling{}.Install(s.Handler.NonGoRestfulMux)
		if c.EnableContentionProfiling {
			runtime.SetBlockProfileRate(1)
		}
	}
	if c.EnableMetrics {
		routes.DefaultMetrics{}.Install(s.Handler.NonGoRestfulMux)
	}
	routes.Version{Version: c.Version}.Install(s.Handler.GoRestfulContainer)
}
