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
	"net/http"
	"strings"

	restful "github.com/emicklei/go-restful"

	"github.com/baidu/openless/pkg/util/logs"

	"github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/server/mux"
)

// ServerHandler holds the different http.Handlers used by the API server.
// This includes the full handler chain, the director (which chooses between gorestful and nonGoRestful,
// the gorestful handler (used for the API) which falls through to the nonGoRestful handler on unregistered paths,
// and the nonGoRestful handler (which can contain a fallthrough of its own)
// FullHandlerChain -> Director -> {GoRestfulContainer,NonGoRestfulMux} based on inspection of registered web services
type ServerHandler struct {
	// FullHandlerChain is the one that is eventually served with.  It should include the full filter
	// chain and then call the Director.
	FullHandlerChain http.Handler

	// The registered APIs.  InstallAPIs uses this.  Other servers probably shouldn't access this directly.
	GoRestfulContainer *restful.Container

	// NonGoRestfulMux is the final HTTP handler in the chain.
	// It comes after all filters and the API handling
	// This is where other servers can attach handler to various parts of the chain.
	NonGoRestfulMux *mux.PathRecorderMux

	// Director is here so that we can properly handle fall through and proxy cases.
	Director http.Handler
}

// HandlerChainBuilderFn is used to wrap the GoRestfulContainer handler using the provided handler chain.
// It is normally used to apply filtering like authentication and authorization
type HandlerChainBuilderFn func(apiHandler http.Handler) http.Handler

// NewServerHandler xxx
func NewServerHandler(name string, handlerChainBuilder HandlerChainBuilderFn, notFoundHandler http.Handler) *ServerHandler {
	nonGoRestfulMux := mux.NewPathRecorderMux(name)
	if notFoundHandler != nil {
		nonGoRestfulMux.NotFoundHandler(notFoundHandler)
	}

	gorestfulContainer := restful.NewContainer()
	gorestfulContainer.ServeMux = http.NewServeMux()
	gorestfulContainer.Router(restful.CurlyRouter{})
	director := director{
		name:               name,
		goRestfulContainer: gorestfulContainer,
		nonGoRestfulMux:    nonGoRestfulMux,
	}
	return &ServerHandler{
		FullHandlerChain:   handlerChainBuilder(director),
		GoRestfulContainer: gorestfulContainer,
		NonGoRestfulMux:    nonGoRestfulMux,
		Director:           director,
	}
}

type director struct {
	name               string
	goRestfulContainer *restful.Container
	nonGoRestfulMux    *mux.PathRecorderMux
}

func (d director) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	req.Header.Add(api.AppNameKey, d.name)
	// check to see if our webservices want to claim this path
	for _, ws := range d.goRestfulContainer.RegisteredWebServices() {
		switch {
		case ws.RootPath() == "/apis":
			// if we are exactly /apis or /apis/, then we need special handling in loop.
			// normally these are passed to the nonGoRestfulMux, but if discovery is enabled, it will go directly.
			// We can't rely on a prefix match since /apis matches everything (see the big comment on Director above)
			if path == "/apis" || path == "/apis/" {
				logs.NewLogger().V(10).WithField(api.HeaderXRequestID, req.Header.Get(api.HeaderXRequestID)).WithField(api.AppNameKey, d.name).Debugf("[msg:%s %q satisfied by gorestful with webservice %v]", req.Method, path, ws.RootPath())
				// logs.Debugf("[%s:%s][%s:%s][msg:%s %q satisfied by gorestful with webservice %v]", api.HeaderXRequestID, req.Header.Get(api.HeaderXRequestID), api.AppNameKey, d.name, req.Method, path, ws.RootPath())
				// don't use servemux here because gorestful servemuxes get messed up when removing webservices
				// TODO fix gorestful, remove TPRs, or stop using gorestful
				d.goRestfulContainer.Dispatch(w, req)
				return
			}

		case strings.HasPrefix(path, ws.RootPath()):
			// ensure an exact match or a path boundary match
			if len(path) == len(ws.RootPath()) || path[len(ws.RootPath())] == '/' {
				logs.NewLogger().V(10).WithField(api.HeaderXRequestID, req.Header.Get(api.HeaderXRequestID)).WithField(api.AppNameKey, d.name).Debugf("[msg:%s %q satisfied by gorestful with webservice %v]", req.Method, path, ws.RootPath())
				// don't use servemux here because gorestful servemuxes get messed up when removing webservices
				// TODO fix gorestful, remove TPRs, or stop using gorestful
				d.goRestfulContainer.Dispatch(w, req)
				return
			}
		}
	}

	// if we didn't find a match, then we just skip gorestful altogether
	logs.V(8).Infof("%v: %v %q satisfied by nonGoRestful", d.name, req.Method, path)
	d.nonGoRestfulMux.ServeHTTP(w, req)
}

// ServeHTTP makes it an http.Handler
func (s *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.FullHandlerChain.ServeHTTP(w, r)
}
