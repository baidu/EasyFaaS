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

package stubs

import (
	"net/http"

	routing "github.com/qiangxue/fasthttp-routing"

	"github.com/baidu/openless/cmd/stubs/options"
)

func HelloWorldHandler(c *routing.Context) error {
	c.SetStatusCode(http.StatusOK)
	c.WriteString("hello controller stubs")
	return nil
}

func InstallHandler(stubsOptions *options.StubsOptions) *routing.Router {
	router := routing.New()
	router.Get("/hello", HelloWorldHandler)

	installApiserver(router, stubsOptions)
	return router
}
