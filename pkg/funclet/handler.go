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

package funclet

import (
	"net/http"
	"time"

	"github.com/baidu/easyfaas/pkg/util/id"

	"github.com/emicklei/go-restful"

	"github.com/baidu/easyfaas/pkg/api"
	svcErr "github.com/baidu/easyfaas/pkg/error"
	funcletCtx "github.com/baidu/easyfaas/pkg/funclet/context"
	"github.com/baidu/easyfaas/pkg/server"
	"github.com/baidu/easyfaas/pkg/server/endpoint"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

func (f *Funclet) NewContext(requestID string, logger *logs.Logger) *funcletCtx.Context {
	if requestID == "" {
		requestID = id.GetRequestID()
	}
	return &funcletCtx.Context{
		RequestID: requestID,
		Logger:    logger.WithField("request_id", requestID),
	}
}

// InstallAPI xxx
func (f *Funclet) InstallAPI(container *restful.Container) {
	var apis = []endpoint.ApiSingle{
		{
			Verb:    "GET",
			Path:    "hello",
			Handler: server.WrapRestRouteFunc(f.HelloWorldHandler),
		},
		{
			Verb:    "POST",
			Path:    "funclet/warmup",
			Handler: server.WrapRestRouteFunc(f.WarmUpHandler),
		},
		{
			Verb:    "POST",
			Path:    "funclet/ide-warmup",
			Handler: server.WrapRestRouteFunc(f.IDEWarmUpHandler),
		},
		{
			Verb:    "POST",
			Path:    "funclet/cooldown",
			Handler: server.WrapRestRouteFunc(f.CoolDownHandler),
		},
		{
			Verb:    "POST",
			Path:    "funclet/reborn",
			Handler: server.WrapRestRouteFunc(f.RebornHandler),
		},
		{
			Verb:    "GET",
			Path:    "funclet/list",
			Handler: server.WrapRestRouteFunc(f.ContainerListHandler),
		},
		{
			Verb:    "GET",
			Path:    "funclet/container/{ContainerID}",
			Handler: server.WrapRestRouteFunc(f.ContainerInfoHandler),
		},
		{
			Verb:    "POST",
			Path:    "funclet/reset",
			Handler: server.WrapRestRouteFunc(f.ResetHandler),
		},
		{
			Verb:    "GET",
			Path:    "funclet/node",
			Handler: server.WrapRestRouteFunc(f.GetNodeHandler),
		},
	}
	var apiversion = []endpoint.ApiVersion{
		{
			Prefix: "/v1",
			Group:  apis,
		},
	}

	endpoint.NewApiInstaller(apiversion).Install(container)
}

func (f *Funclet) HelloWorldHandler(c *server.Context) {
	c.Response().WriteHeaderAndEntity(http.StatusOK, "hello funclet")
}

func (f *Funclet) ContainerListHandler(c *server.Context) {
	response := c.Response()
	logger := c.Logger()

	logger.V(6).Infof("Get container list")
	defer logger.TimeTrack(time.Now(), "Get container list finish")

	criteria := api.NewListContainerCriteria()
	criteria.ReadFromRequest(c.HTTPRequest())
	containerList, err := f.List(criteria)
	if err != nil {
		response.WriteErrorString(http.StatusInternalServerError, err.Error())
		c.WithErrorLog(err).WriteTo(response)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, containerList)
}

func (f *Funclet) ContainerInfoHandler(c *server.Context) {
	request := c.Request()
	response := c.Response()
	logger := c.Logger()
	logger.V(6).Infof("Get container info")
	defer logger.TimeTrack(time.Now(), "Get container info finish")

	containerID := request.PathParameter("ContainerID")
	if len(containerID) == 0 {
		err := svcErr.NewInvalidParameterValueException("get container from url path", nil)
		c.WithErrorLog(err).WriteTo(response)
		return
	}
	containerInfo, err := f.ContainerInfo(containerID)
	if err != nil {
		response.WriteErrorString(http.StatusInternalServerError, err.Error())
		c.WithErrorLog(err).WriteTo(response)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, containerInfo)
}

func (f *Funclet) WarmUpHandler(c *server.Context) {
	response := c.Response()
	logger := c.Logger()

	params := api.WarmupRequest{}
	if err := c.Request().ReadEntity(&params); err != nil {
		c.WithWarnLog(err).WriteTo(response)
		return
	}

	logger.Infof("warm up container %s start", params.ContainerID)
	defer logger.TimeTrack(time.Now(), "warm up container finish")
	fCtx := f.NewContext(c.RequestID(), logger)
	if err := f.WarmUpContainerEvent(fCtx, params); err != nil {
		c.WithErrorLog(err).WriteTo(response)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, api.WarmUpResponse{Container: *fCtx.Container})
}

func (f *Funclet) IDEWarmUpHandler(c *server.Context) {
	response := c.Response()
	logger := c.Logger()

	params := api.WarmupRequest{}
	if err := c.Request().ReadEntity(&params); err != nil {
		c.WithWarnLog(err).WriteTo(response)
		return
	}

	logger.Infof("warm up container %s start", params.ContainerID)
	defer logger.TimeTrack(time.Now(), "warm up container finish")
	fCtx := f.NewContext(c.RequestID(), logger)
	if err := f.IDEWarmUpContainerEvent(fCtx, params); err != nil {
		c.WithErrorLog(err).WriteTo(response)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, api.WarmUpResponse{Container: *fCtx.Container})
}

func (f *Funclet) CoolDownHandler(c *server.Context) {
	response := c.Response()
	logger := c.Logger()

	params := api.ResetRequest{}
	if err := c.Request().ReadEntity(&params); err != nil {
		c.WithWarnLog(err).WriteTo(response)
		return
	}
	logger.V(6).Infof("cool down container %s start", params.ContainerID)
	defer logger.TimeTrack(time.Now(), "cool down container finish")
	fCtx := f.NewContext(c.RequestID(), logger)
	err, res := f.ResetContainerEvent(fCtx, &params)
	if err != nil {
		c.WithErrorLog(err).WriteTo(response)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, res)
}

func (f *Funclet) RebornHandler(c *server.Context) {
	response := c.Response()
	logger := c.Logger()

	params := api.ResetRequest{}
	if err := c.Request().ReadEntity(&params); err != nil {
		c.WithWarnLog(err).WriteTo(response)
		return
	}
	logger.V(6).Infof("reborn container %s start ", params.ContainerID)
	defer logger.TimeTrack(time.Now(), "reborn container finish")
	fCtx := f.NewContext(c.RequestID(), logger)
	err, res := f.ResetContainerEvent(fCtx, &params)
	if err != nil {
		c.WithErrorLog(err).WriteTo(response)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, res)
}

func (f *Funclet) ResetHandler(c *server.Context) {
	response := c.Response()
	logger := c.Logger()

	logger.V(6).Infof("reset node start")
	defer logger.V(6).TimeTrack(time.Now(), "reset node finish")

	err := f.Reset()
	if err != nil {
		response.WriteErrorString(http.StatusInternalServerError, err.Error())
		c.WithErrorLog(err).WriteTo(response)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, "")
}

func (f *Funclet) GetNodeHandler(c *server.Context) {
	response := c.Response()
	logger := c.Logger()

	logger.V(6).Infof("get node start")
	defer logger.TimeTrack(time.Now(), "get node finish")
	info, err := f.NodeInfo()
	if err != nil {
		response.WriteErrorString(http.StatusInternalServerError, err.Error())
		c.WithErrorLog(err).WriteTo(response)
		return
	}
	response.WriteHeaderAndEntity(http.StatusOK, info)
}
