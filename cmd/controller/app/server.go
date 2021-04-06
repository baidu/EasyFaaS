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

package app

import (
	"fmt"
	"net/http/pprof"
	"strings"

	"github.com/baidu/openless/pkg/controller/client"

	"github.com/baidu/openless/pkg/httptrigger"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	routing "github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/baidu/openless/pkg/util/logs"

	"github.com/baidu/openless/cmd/controller/options"
	"github.com/baidu/openless/pkg/controller"
)

func Run(runOptions *options.ControllerOptions) error {
	port := runOptions.RecommendedOptions.SecureServing.BindPort
	addr := fmt.Sprintf(":%d", port)

	app, err := Init(runOptions)
	if err != nil {
		return err
	}
	handler := wrapRouter(app, runOptions)
	return fasthttp.ListenAndServe(addr, handler)
}

func Init(runOptions *options.ControllerOptions) (app *controller.Controller, err error) {
	app, err = controller.Init(runOptions)
	if err != nil {
		return nil, err
	}
	if runOptions.HTTPEnhanced {
		c := client.NewControllerCallClient(app, app.FuncletClient, runOptions)
		httptrigger.InitWithClient(c)
	}
	return
}

func wrapRouter(controller *controller.Controller, runOptions *options.ControllerOptions) func(ctx *fasthttp.RequestCtx) {
	router := routing.New()
	router.Get("/healthz", controller.HealthzHandler)

	router.Post("/v1/functions/<functionName>/invocations", controller.InvokeHandler)
	router.Get("/v1/runtimes", controller.ListRuntimesHandler)
	router.Get("/v1/resource", controller.GetResourceHandler)
	router.Post("/v1/runtimes/<runtimeID>/invalidate", controller.InvalidateRuntime)

	if runOptions.HTTPEnhanced {
		logs.V(9).Info("equipped with http trigger feature")
		router.Any(`/<userID:\w+>/<functionName>`, httptrigger.ProxyHandler)
		router.Any(`/<userID:\w+>/<functionName>/<version>`, httptrigger.ProxyHandler)
	}

	var enhanceRouting bool
	if runOptions.RecommendedOptions.Features.EnableMetrics || runOptions.RecommendedOptions.Features.EnableProfiling {
		enhanceRouting = true
	}
	logs.Infof("enhanceRouting flag %v", enhanceRouting)
	if !enhanceRouting {
		return router.HandleRequest
	}

	enhanceHandler := func(ctx *fasthttp.RequestCtx) {
		urlPath := string(ctx.Path())
		if runOptions.RecommendedOptions.Features.EnableProfiling && strings.HasPrefix(urlPath, "/debug/pprof/") {
			logs.V(9).Info("enable profiling")
			switch urlPath {
			case "/debug/pprof/cmdline":
				fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Cmdline)(ctx)
			case "/debug/pprof/profile":
				fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Profile)(ctx)
			case "/debug/pprof/symbol":
				fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Symbol)(ctx)
			case "/debug/pprof/trace":
				fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Trace)(ctx)
			default:
				fasthttpadaptor.NewFastHTTPHandlerFunc(pprof.Index)(ctx)
			}
			return
		}

		if runOptions.RecommendedOptions.Features.EnableMetrics && strings.HasPrefix(urlPath, "/metrics") {
			logs.V(9).Info("enable metrics")
			fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())(ctx)
			return
		}

		router.HandleRequest(ctx)
		return
	}
	return enhanceHandler
}
