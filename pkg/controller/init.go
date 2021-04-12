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

package controller

import (
	"time"

	"github.com/baidu/easyfaas/pkg/controller/function"

	"github.com/baidu/easyfaas/pkg/funclet/client"

	"github.com/baidu/easyfaas/pkg/util/id"

	"github.com/baidu/easyfaas/cmd/controller/options"
	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/controller/rtctrl"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

func Init(options *options.ControllerOptions) (controller *Controller, err error) {
	controller = &Controller{
		runOptions:    options,
		FuncletClient: client.NewFuncletClient(options.FuncletClientOptions),
	}

	for {
		err := controller.initContainers()
		if err != nil {
			logs.Errorf("get container failed: %s", err.Error())
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	controller.runtimeControl, err = rtctrl.NewRuntimeClient(options.RuntimeConfigOptions,
		options.DispatcherV2Options, controller.runtimeDispatcher)
	if err != nil {
		return nil, err
	}
	cacheConfig := function.DefaultCacheExpirationConfigs()
	cacheConfig[function.CacheTypeAlias] = options.AliasCacheOptions.CacheExpiration
	controller.dataStorer, err = function.NewDataStorer(options.RepositoryOptions, function.NewStorageCache(cacheConfig))
	insideRepositoryOptions := options.RepositoryOptions
	insideRepositoryOptions.Version = "inside-v1"
	controller.insideDataStorer, err = function.NewDataStorer(insideRepositoryOptions, function.NewStorageCache(cacheConfig))
	if options.HTTPEnhanced {
		if options.HttpTriggerRepositoryOptions.IsEmpty() {
			controller.httpTriggerDataStorer, err = function.NewDataStorer(insideRepositoryOptions, function.NewStorageCache(cacheConfig))
		} else {
			controller.httpTriggerDataStorer, err = function.NewDataStorer(options.HttpTriggerRepositoryOptions, function.NewStorageCache(cacheConfig))
		}
	}
	if err != nil {
		return nil, err
	}
	go controller.cronTask(options)
	if options.RecommendedOptions.Features.EnableMetrics {
		go controller.metricTask(options)
	}
	return
}

func (controller *Controller) initContainers() (err error) {
	// TODO: sync from funclet for more runtime information
	nodeInfo, err := controller.FuncletClient.NodeInfo(&api.FuncletClientListNodeInput{RequestID: id.GetRequestID()})
	if err != nil {
		return err
	}
	params := &rtctrl.RuntimeManagerParameters{
		MaxRuntimeIdle:        controller.runOptions.MaxRuntimeIdle,
		MaxRunnerDefunct:      controller.runOptions.MaxRunnerDefunct,
		MaxRunnerResetTimeout: controller.runOptions.MaxRunnerResetTimeout,
	}
	controller.runtimeDispatcher = rtctrl.NewRuntimeManager(nodeInfo, params)

	res, err := controller.FuncletClient.List(&api.FuncletClientListContainersInput{RequestID: id.GetRequestID()})
	if err != nil {
		return err
	}
	allContainers := *res
	logs.Infof("get container %+v", allContainers)

	for _, container := range allContainers {
		// set default runtime concurrent mode from service option
		params := &rtctrl.NewRuntimeParameters{
			RuntimeID:               container.ContainerID,
			ConcurrentMode:          controller.runOptions.ConcurrentMode,
			StreamMode:              container.WithStreamMode,
			WaitRuntimeAliveTimeout: controller.runOptions.RuntimeConfigOptions.WaitRuntimeAliveTimeout,
			IsFrozen:                container.IsFrozen,
			Resource:                container.Resource,
		}
		controller.runtimeDispatcher.NewRuntime(params)
	}
	return nil
}
