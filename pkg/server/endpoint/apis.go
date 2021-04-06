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

package endpoint

import (
	restful "github.com/emicklei/go-restful"

	"github.com/baidu/openless/pkg/util/logs"
)

type ApiVersion struct {
	Prefix string
	Group  []ApiSingle
}

type ApiSingle struct {
	Verb    string
	Path    string
	Handler restful.RouteFunction
	Filters []restful.FilterFunction
}

func (v ApiVersion) InstallREST(container *restful.Container) {
	logs.V(5).Infof("v.Prefix=%s", v.Prefix)
	ws := new(restful.WebService)
	ws.Path(v.Prefix).
		Consumes("*/*").
		Produces(restful.MIME_JSON)

	for _, api := range v.Group {
		rb := ws.Method(api.Verb).Path(api.Path).To(api.Handler)
		if api.Filters != nil {
			for _, filter := range api.Filters {
				rb.Filter(filter)
			}
		}
		ws.Route(rb)
	}
	container.Add(ws)
}
