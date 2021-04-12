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
	"sync"
	"time"

	"github.com/baidu/easyfaas/pkg/util/id"
	"github.com/baidu/easyfaas/pkg/util/logs/metric"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/baidu/easyfaas/pkg/util/logs"

	"github.com/baidu/easyfaas/cmd/controller/options"
	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/controller/rtctrl"
)

// cronTask: response to gc and health check
// TODO: Should move into runtime scheduler
func (controller *Controller) cronTask(opt *options.ControllerOptions) {
	taskID := id.GetTaskID()
	logger := logs.NewLogger().WithField("task_id", taskID)
	interval := time.Duration(opt.TaskInterval) * time.Second
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			logger.Debug("start cron task")
			controller.resourceTask(logger)
			controller.runtimeTask(logger)
			logger.Debug("finish cron task")
		}
	}
}

func (controller *Controller) metricTask(opt *options.ControllerOptions) {
	taskID := id.GetTaskID()
	logger := logs.NewLogger().WithField("metric_task_id", taskID)
	interval := time.Duration(opt.MetricsTaskInterval) * time.Second
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			logger.Debug("start metric task")
			controller.overallMetrics()
			controller.runtimeMetrics(logger)
			logger.Debug("finish metric task")
		}
	}
}

func (controller *Controller) resourceTask(logger *logs.Logger) {
	logger.Info("start resource task")
	nodeInfo, err := controller.FuncletClient.NodeInfo(&api.FuncletClientListNodeInput{RequestID: id.GetRequestID()})
	if err != nil {
		logger.Errorf("sync resource info from funclet failed: %s", err)
		return
	}
	if nodeInfo.Resource != nil {
		if !controller.runtimeDispatcher.SyncResource(nodeInfo.Resource) {
			logger.Errorf("sync resource info failed: %s", err)
			return
		}
	}
	list, err := controller.FuncletClient.List(&api.FuncletClientListContainersInput{RequestID: id.GetRequestID()})
	if err != nil {
		logger.Errorf("get container list from funclet failed: %s", err)
		return
	}
	var wg sync.WaitGroup
	for _, rt := range *list {
		wg.Add(1)
		go func(runtime *api.ContainerInfo) {
			if syncResult, err := controller.runtimeDispatcher.SyncRuntimeResource(runtime.ContainerID, runtime.Resource); !syncResult {
				if err != nil {
					inUse := rtctrl.RuntimeSyncError{RuntimeID: runtime.ContainerID, Reason: "runtime in used"}.Error()
					if err.Error() != inUse {
						logs.Errorf("sync runtime %s resource failed: resource %s", runtime.ContainerID, runtime.Resource)
					}
				}
			}
			wg.Done()
		}(rt)
	}
	wg.Wait()
	logger.Info("finish resource task")
}

func (controller *Controller) runtimeTask(logger *logs.Logger) {
	logger.Info("start runtime task")
	runtimes := controller.runtimeDispatcher.RuntimeList()
	var wg sync.WaitGroup
	for _, runtime := range runtimes {
		wg.Add(1)
		go func(rt *rtctrl.RuntimeInfo, logger *logs.Logger) {
			controller.processRuntime(rt, logger)
			wg.Done()
		}(runtime, logger)
	}
	wg.Wait()
	logger.Info("finish runtime task")
}

func (controller *Controller) processRuntime(runtime *rtctrl.RuntimeInfo, logger *logs.Logger) {
	controller.coolDown(runtime, logger)
	controller.reborn(runtime, logger)
}

// coolDown
func (controller *Controller) coolDown(runtime *rtctrl.RuntimeInfo, logger *logs.Logger) {
	recommend, err := controller.runtimeDispatcher.CoolDownRuntime(runtime)
	if err != nil {
		return
	}
	res := runtime.Resource.Copy()
	used := runtime.Used
	marked := runtime.Marked

	runtime.RebootBegin()
	defer runtime.RebootEnd()

	if recommend != nil {
		for _, ID := range recommend.ResetContainers {
			info, _ := controller.runtimeDispatcher.GetRuntime(ID)
			info.RebootBegin()
		}
		defer func() {
			for _, ID := range recommend.ResetContainers {
				info, _ := controller.runtimeDispatcher.GetRuntime(ID)
				info.RebootEnd()
			}
		}()
	}

	requestID := id.GetRequestID()
	response, err := controller.FuncletClient.CoolDown(&api.FuncletClientCoolDownInput{
		ContainerID:             runtime.RuntimeID,
		RequestID:               requestID,
		ScaleDownRecommendation: recommend,
	})
	logger.Infof("cool down runtime %s request id %s response %v err %v", runtime.RuntimeID, requestID, response, err)
	if err == nil {
		logger.V(9).Infof("[resource modify]-[decrease]: used %t, runtime %s, resource %s", used, runtime.RuntimeID, res)
		if used {
			if !controller.runtimeDispatcher.ReleaseUsedResource(res) {
				logs.Errorf("cool down runtime %s release memory %d failed", runtime.RuntimeID, runtime.Resource.Memory)
			} else {
				runtime.SetUsed(false)
			}
		}
		if marked {
			if !controller.runtimeDispatcher.ReleaseMarkedResource(res) {
				logger.Errorf("cool down runtime %s release memory %d failed", runtime.RuntimeID, runtime.Resource.Memory)
			} else {
				runtime.SetMarked(false)
			}
		}
	}
	if err != nil {
		logger.Errorf("cool down runtime %s failed: %s", runtime.RuntimeID, err.Error())
		runtime.Invalidate()
	}
	if recommend != nil && response == nil {
		for _, ID := range recommend.ResetContainers {
			rt, _ := controller.runtimeDispatcher.GetRuntime(ID)
			rt.Invalidate()
		}
		logger.Errorf("scale down runtime %s failed: %v", runtime.RuntimeID, recommend.ResetContainers)
	}
	if response != nil && response.ScaleDownResult != nil && len(response.ScaleDownResult.Fails) != 0 {
		for _, item := range response.ScaleDownResult.Fails {
			rt, _ := controller.runtimeDispatcher.GetRuntime(item.ContainerID)
			rt.Invalidate()
		}
		logger.Errorf("scale down runtime %s failed: %v", runtime.RuntimeID, response.ScaleDownResult.Fails)
	}
}

func (controller *Controller) reborn(runtime *rtctrl.RuntimeInfo, logger *logs.Logger) {
	recommend, err := controller.runtimeDispatcher.ResetRuntime(runtime)
	if err != nil {
		return
	}
	res := runtime.Resource.Copy()
	used := runtime.Used
	marked := runtime.Marked

	runtime.RebootBegin()
	defer runtime.RebootEnd()

	if recommend != nil {
		for _, ID := range recommend.ResetContainers {
			info, _ := controller.runtimeDispatcher.GetRuntime(ID)
			info.RebootBegin()
		}
		defer func() {
			for _, ID := range recommend.ResetContainers {
				info, _ := controller.runtimeDispatcher.GetRuntime(ID)
				info.RebootEnd()
			}
		}()
	}

	requestID := id.GetRequestID()
	response, err := controller.FuncletClient.Reborn(&api.FuncletClientRebornInput{
		ContainerID:             runtime.RuntimeID,
		RequestID:               requestID,
		ScaleDownRecommendation: recommend,
	})
	logger.Infof("reborn runtime %s request id %s response %v err %v", runtime.RuntimeID, requestID, response, err)
	if err == nil {
		logs.V(9).Infof("[resource modify]-[decrease]: used %t, runtime %s, resource %s", used, runtime.RuntimeID, res)
		if used {
			if !controller.runtimeDispatcher.ReleaseUsedResource(res) {
				logs.Errorf("cool down runtime %s release memory %d failed", runtime.RuntimeID, runtime.Resource.Memory)
			} else {
				runtime.SetUsed(false)
			}
		}
		if marked {
			if !controller.runtimeDispatcher.ReleaseMarkedResource(res) {
				logs.Errorf("cool down runtime %s release memory %d failed", runtime.RuntimeID, runtime.Resource.Memory)
			} else {
				runtime.SetMarked(false)
			}
		}
	}
	if err != nil {
		logger.Errorf("reborn runtime %s failed: %s", runtime.RuntimeID, err.Error())
	}
	if recommend != nil && response == nil {
		for _, ID := range recommend.ResetContainers {
			rt, _ := controller.runtimeDispatcher.GetRuntime(ID)
			rt.Invalidate()
		}
		logger.Errorf("scale down runtime %s failed: %v", runtime.RuntimeID, recommend.ResetContainers)
	}
	if response != nil && response.ScaleDownResult != nil && len(response.ScaleDownResult.Fails) != 0 {
		for _, item := range response.ScaleDownResult.Fails {
			rt, _ := controller.runtimeDispatcher.GetRuntime(item.ContainerID)
			rt.Invalidate()
		}
		logger.Errorf("scale down runtime %s failed: %v", runtime.RuntimeID, response.ScaleDownResult.Fails)
	}
}

func (controller *Controller) overallMetrics() {
	if !controller.runOptions.RecommendedOptions.Features.EnableMetrics {
		return
	}
	cold, used, all := controller.runtimeDispatcher.RuntimeStatistics()
	metric.SetGauge(RuntimeMetricName(MetricRuntimeCold), float64(cold))
	metric.SetGauge(RuntimeMetricName(MetricRuntimeInUse), float64(used))
	metric.SetGauge(RuntimeMetricName(MetricRuntimeAll), float64(all))
}

func (controller *Controller) runtimeMetrics(logger *logs.Logger) {
	if !controller.runOptions.RecommendedOptions.Features.EnableMetrics {
		return
	}
	logger.Info("start runtime stats task")
	runtimes := controller.runtimeDispatcher.RuntimeList()
	var wg sync.WaitGroup
	for _, runtime := range runtimes {
		if runtime == nil {
			continue
		}
		wg.Add(1)
		go func(rt *rtctrl.RuntimeInfo, logger *logs.Logger) {
			controller.processRuntimeStats(rt, logger)
			wg.Done()
		}(runtime, logger)
	}
	wg.Wait()
	logger.Info("finish runtime stats task")
}

func (controller *Controller) processRuntimeStats(rt *rtctrl.RuntimeInfo, logger *logs.Logger) {
	input := api.FuncletClientContainerInfoInput{
		RequestID: id.GetRequestID(),
		ID:        rt.RuntimeID,
	}
	info, err := controller.FuncletClient.Info(&input)
	if err != nil {
		logger.Errorf("get runtime %s info from funclet failed: %s", rt.RuntimeID, err)
		return
	}
	if info.ResourceStats == nil || info.ResourceStats.CPUStats == nil || info.ResourceStats.MemoryStats == nil {
		logger.Warnf("get runtime %s info from funclet failed: resource stats is nil", rt.RuntimeID)
		return
	}
	value := metric.GetCounterValue(RuntimeMetricName(RuntimeCPUUsageSeconds), info.ContainerID)
	inc := (float64(info.ResourceStats.CPUStats.TotalUsage) - value) / float64(time.Second)
	metric.AddWithLabels(RuntimeMetricName(RuntimeCPUUsageSeconds), inc, getResourceLabels(info.ContainerID))
	metric.SetGaugeWithLabels(RuntimeMetricName(RuntimeMemoryUsageBytes), float64(info.ResourceStats.MemoryStats.Usage), getResourceLabels(info.ContainerID))
	return
}

func getResourceLabels(ID string) *prometheus.Labels {
	labels := prometheus.Labels{}
	labels[RuntimeIDLabel] = ID
	return &labels
}
