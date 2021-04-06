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
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/baidu/openless/pkg/controller/rtctrl"
	"github.com/baidu/openless/pkg/util/logs"
	"github.com/baidu/openless/pkg/util/logs/metric"
)

const (
	getFunctionBrnLabel           = "function_brn"
	getFunctionHitCacheLabel      = "get_function_hit_cache"
	getConfigurationHitCacheLabel = "get_configuration_hit_cache"
	podSourceLabel                = "pod_source"
	responseCodeLabel             = "response_code"
)

var allCostLabelList = []string{getFunctionHitCacheLabel, getConfigurationHitCacheLabel, podSourceLabel, responseCodeLabel, getFunctionBrnLabel}

var (
	stages = []metric.MetricConfig{
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageGetFunction),
			Name:         StageName(StageGetFunction),
			Labels:       []string{},
			HelpTemplate: "get function meta from apiserver %s",
			Buckets:      []float64{50000, 100000, 200000},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageGetFunctionHitCache),
			Name:         StageName(StageGetFunctionHitCache),
			Labels:       []string{},
			HelpTemplate: "get function meta from cache %s",
			Buckets:      []float64{1, 3, 5, 8, 10, 20, 50, 500, 1000, 10000},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageGetRuntimeConfiguration),
			Name:         StageName(StageGetRuntimeConfiguration),
			Labels:       []string{},
			HelpTemplate: "get runtime configuration meta from apiserver %s",
			Buckets:      []float64{20000, 30000, 40000, 50000, 200000},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageGetRuntimeConfigurationHitCache),
			Name:         StageName(StageGetRuntimeConfigurationHitCache),
			Labels:       []string{},
			HelpTemplate: "get runtime configuration meta from cache %s",
			Buckets:      []float64{0.01, 0.05, 0.1, 0.5, 0.8, 1, 5, 10},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageGetPodWarm),
			Name:         StageName(StageGetPodWarm),
			Labels:       []string{},
			HelpTemplate: "get warm runtime info from funclet or cache %s",
			Buckets:      []float64{1, 2, 3, 6, 8, 10, 20, 30, 50, 200, 500},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageGetPodCold),
			Name:         StageName(StageGetPodCold),
			Labels:       []string{},
			HelpTemplate: "get cold runtime info from funclet or cache %s",
			Buckets:      []float64{2000, 5000, 10000},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StagePutPod),
			Name:         StageName(StagePutPod),
			Labels:       []string{},
			HelpTemplate: "put runtime info to funclet or cache %s",
			Buckets:      []float64{0.01, 0.05, 0.1, 0.5, 1, 5},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageInvocation),
			Name:         StageName(StageInvocation),
			Labels:       []string{},
			HelpTemplate: "invoke %s",
			Buckets:      []float64{500, 1000, 1500, 2000, 5000, 10000, 20000, 50000},
			HasSummary:   true,
		},
	}

	overall = []metric.MetricConfig{
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageAllCost),
			Name:         StageName(StageAllCost),
			Labels:       allCostLabelList,
			HelpTemplate: "All cost of invocation %s",
			Buckets:      []float64{200, 500, 700, 1000, 1500, 2000, 3000, 5000, 10000, 50000},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageStageSumUp),
			Name:         StageName(StageStageSumUp),
			Labels:       []string{},
			HelpTemplate: "Sum cost of all stages %s",
			Buckets:      []float64{200, 500, 1000, 1500, 2000, 3000, 5000, 10000, 50000},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageSchedule),
			Name:         StageName(StageSchedule),
			Labels:       []string{},
			HelpTemplate: "get runtime info from funclet or cache %s",
			Buckets:      []float64{10, 20, 30, 50, 100, 200},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageOther),
			Name:         StageName(StageOther),
			Labels:       []string{},
			HelpTemplate: "Other overhead (go runtime) %s",
			Buckets:      []float64{5, 10, 15, 20, 50, 100, 500, 1000},
			HasSummary:   true,
		},
	}

	statistic = []metric.MetricConfig{
		{
			MetricType:   metric.MetricTypeGauge,
			Index:        RuntimeMetricName(MetricRuntimeCold),
			Name:         RuntimeMetricName(MetricRuntimeCold),
			Labels:       []string{},
			HelpTemplate: "the count of cold runtime",
			HasSummary:   false,
		},
		{
			MetricType:   metric.MetricTypeGauge,
			Index:        RuntimeMetricName(MetricRuntimeInUse),
			Name:         RuntimeMetricName(MetricRuntimeInUse),
			Labels:       []string{},
			HelpTemplate: "the count of in used runtime",
			HasSummary:   false,
		},
		{
			MetricType:   metric.MetricTypeGauge,
			Index:        RuntimeMetricName(MetricRuntimeAll),
			Name:         RuntimeMetricName(MetricRuntimeAll),
			Labels:       []string{},
			HelpTemplate: "the sum count of runtime",
			HasSummary:   false,
		},
		{
			MetricType:   metric.MetricTypeGauge,
			Index:        RuntimeMetricName(RuntimeMemoryUsageBytes),
			Name:         RuntimeMetricName(RuntimeMemoryUsageBytes),
			Labels:       RuntimeResourceLabels,
			HelpTemplate: "memory usage of runtime(bytes)",
			HasSummary:   false,
		},
		{
			MetricType:   metric.MetricTypeCounter,
			Index:        RuntimeMetricName(RuntimeCPUUsageSeconds),
			Name:         RuntimeMetricName(RuntimeCPUUsageSeconds),
			Labels:       RuntimeResourceLabels,
			HelpTemplate: "cpu usage of runtime(seconds)",
			HasSummary:   false,
		},
	}
)

type metricStage = int

const (
	StageGetFunction metricStage = iota
	StageGetFunctionHitCache
	StageGetRuntimeConfiguration
	StageGetRuntimeConfigurationHitCache
	StageGetPodWarm
	StageGetPodCold
	StagePutPod
	StageInvocation
	StageAllCost
	StageStageSumUp
	StageSchedule
	StageOther
)

var (
	stageStr = [StageOther + 1]string{
		"get_function",
		"get_function_hit_cache",
		"get_runtime_configuration",
		"get_runtime_configuration_hit_cache",
		"get_pod_warm",
		"get_pod_cold",
		"put_pod",
		"invoke",
		"all_cost",
		"stage_sum_up_cost",
		"schedule_overhead_cost",
		"go_runtime_overhead_cost",
	}
)

func StageName(ms metricStage) string {
	return stageStr[ms]
}

type metricRuntime = int

const (
	ResourceTypeLabel = "resource_type"
	RuntimeIDLabel    = "runtime_id"
)

var RuntimeResourceLabels = []string{RuntimeIDLabel}

const (
	MetricRuntimeInUse metricRuntime = iota
	MetricRuntimeCold
	MetricRuntimeAll
	RuntimeMemoryUsageBytes
	RuntimeCPUUsageSeconds
)

var (
	runtimeStr = [RuntimeCPUUsageSeconds + 1]string{
		"runtime_in_use",
		"runtime_cold",
		"runtime_all",
		"runtime_usage_memory_bytes",
		"runtime_usage_cpu_seconds",
	}
)

func RuntimeMetricName(m metricRuntime) string {
	return runtimeStr[m]
}

const (
	InvokeErrNoReason     = ""
	InvokeInvalidParam    = "invoke_invalid_param"
	InvokeTooManyRequests = "invoke_too_many_requests"
	InvokeRuntimeError    = "invoke_runtime_error"
)

type invokeStage struct {
	stage    metricStage
	startT   time.Time
	endT     time.Time
	duration time.Duration
	scrap    bool
}

func (i *invokeStage) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("start_time", i.startT.UnixNano())
	enc.AddInt64("end_time", i.endT.UnixNano())
	enc.AddFloat64("cost_ms", float64(i.duration.Nanoseconds())/1e6)

	return nil
}

type InvokeMetrics struct {
	requestID    string
	startT       time.Time
	endT         time.Time
	invokeStages [StageInvocation + 1]*invokeStage
	overall      map[string]time.Duration
	rtCtrl       *rtctrl.RtCtrlInvokeMetric
	allLabels    map[string]string
}

func NewInvokeMetrics(reqID string) *InvokeMetrics {
	im := &InvokeMetrics{
		requestID:    reqID,
		startT:       time.Now(),
		invokeStages: [StageInvocation + 1]*invokeStage{},
		overall:      make(map[string]time.Duration, StageOther-StageInvocation),
		allLabels: map[string]string{
			getFunctionBrnLabel:           "",
			getFunctionHitCacheLabel:      "",
			getConfigurationHitCacheLabel: "",
			podSourceLabel:                "",
			responseCodeLabel:             "",
		},
	}
	for i := 0; i < StageInvocation+1; i++ {
		im.invokeStages[i] = &invokeStage{
			stage: i,
		}
	}
	return im
}

func (i *InvokeMetrics) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("start_time", i.startT.UnixNano())
	enc.AddInt64("end_time", i.endT.UnixNano())

	for key, value := range i.overall {
		enc.AddFloat64(key, float64(value.Nanoseconds())/1e6)
	}

	for stage, item := range i.invokeStages {
		if item.scrap {
			continue
		}

		enc.AddObject(StageName(stage), item)
	}
	if i.rtCtrl != nil {
		enc.AddObject("rtInvoke", i.rtCtrl)
	}

	return nil
}

func (i *InvokeMetrics) ScrapStage(stage metricStage) error {
	if stage > StageInvocation {
		return fmt.Errorf("stage %d not exists", stage)
	}
	if v := i.invokeStages[stage]; v == nil {
		return fmt.Errorf("stage %d not exists", stage)
	} else {
		v.scrap = true
	}
	return nil
}

func (i *InvokeMetrics) SetLabel(k, v string) {
	if _, ok := i.allLabels[k]; !ok {
		// TODO: err handler?
		// not support label
		return
	}

	i.allLabels[k] = v
}

func (i *InvokeMetrics) StepStart(stage metricStage) error {
	if stage > StageInvocation {
		return fmt.Errorf("stage %d not exists", stage)
	}
	if v := i.invokeStages[stage]; v == nil {
		return fmt.Errorf("stage %d not exists", stage)
	} else {
		v.startT = time.Now()
	}
	return nil
}

func (i *InvokeMetrics) StepDone(stage metricStage) error {
	if stage > StageInvocation {
		return fmt.Errorf("stage %d not exists", stage)
	}
	i.invokeStages[stage].endT = time.Now()
	i.invokeStages[stage].duration = i.invokeStages[stage].endT.Sub(i.invokeStages[stage].startT)

	metric.Observe(StageName(stage), float64(i.invokeStages[stage].duration/time.Microsecond))
	return nil
}

func (i *InvokeMetrics) StepDoneWithLabel(stage metricStage, k, v string) error {
	if stage > StageInvocation {
		return fmt.Errorf("stage %d not exists", stage)
	}
	i.invokeStages[stage].endT = time.Now()
	i.invokeStages[stage].duration = i.invokeStages[stage].endT.Sub(i.invokeStages[stage].startT)

	metric.Observe(StageName(stage), float64(i.invokeStages[stage].duration/time.Microsecond), v)
	return nil
}

func (i *InvokeMetrics) Overall() {
	i.endT = time.Now()
	allCost := i.endT.Sub(i.startT)
	var stageSumUp, schedule time.Duration
	for s, m := range i.invokeStages {
		if m.scrap {
			continue
		}

		stageSumUp += m.duration
		if s != StageInvocation {
			schedule += m.duration
		}
	}

	i.overall[StageName(StageAllCost)] = allCost
	i.overall[StageName(StageStageSumUp)] = stageSumUp
	i.overall[StageName(StageSchedule)] = schedule
	i.overall[StageName(StageOther)] = allCost - stageSumUp
	i.observeOverall()

	if i.rtCtrl != nil {
		i.rtCtrl.ObserveAll()
	}

	return
}

func (i *InvokeMetrics) WriteSummary(overheadMs int) {
	// if overhead == 0, record summary log
	// if overhead >= overall["schedule_overhead_cost"], record summary log
	if i.overall[StageName(StageSchedule)] < time.Duration(overheadMs)*time.Millisecond {
		return
	}

	fields := make([]zap.Field, 0, len(i.allLabels))
	for k, v := range i.allLabels {
		if len(v) == 0 {
			continue
		}
		fields = append(fields, zap.String(k, v))
	}

	logs.NewSummaryLogger().With(fields...).Info("summary", zap.Object("cost", i))
}

func (i *InvokeMetrics) observeOverall() {
	for stage, elapsed := range i.overall {
		if stage == StageName(StageAllCost) {
			labels := prometheus.Labels(i.allLabels)
			metric.ObserveWithLabels(stage, float64(elapsed/time.Microsecond), &labels)

			continue
		}

		metric.Observe(stage, float64(elapsed/time.Microsecond))
	}
}

func init() {
	metric.InitMetric("controller")
	if err := metric.Register("invoke", stages); err != nil {
		panic(err)
	}

	if err := metric.Register("invoke", overall); err != nil {
		panic(err)
	}

	if err := metric.Register("invoke", statistic); err != nil {
		panic(err)
	}
	if err := rtctrl.InitRtCtrlMetric(); err != nil {
		panic(err)
	}
}
