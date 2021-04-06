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

package rtctrl

import (
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/baidu/openless/pkg/util/logs/metric"
)

var (
	stages = []metric.MetricConfig{
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageWaitRuntime),
			Name:         StageName(StageWaitRuntime),
			Labels:       []string{},
			HelpTemplate: "wait runtime %s",
			Buckets:      []float64{1, 2, 5, 10, 30},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageStartRecvLog),
			Name:         StageName(StageStartRecvLog),
			Labels:       []string{},
			HelpTemplate: "start receive log %s",
			Buckets:      []float64{1, 2, 5, 10, 30},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageInvokeFunc),
			Name:         StageName(StageInvokeFunc),
			Labels:       []string{},
			HelpTemplate: "send invoke func request %s",
			Buckets:      []float64{10, 50, 100, 200, 500, 1000, 5000},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageSendRequest),
			Name:         StageName(StageSendRequest),
			Labels:       []string{},
			HelpTemplate: "invoke send request %s",
			Buckets:      []float64{10, 50, 100, 200, 500, 1000, 5000},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageRecvResponse),
			Name:         StageName(StageRecvResponse),
			Labels:       []string{},
			HelpTemplate: "invoke receive response %s",
			Buckets:      []float64{10, 50, 100, 200, 500, 1000, 5000},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageInvokeDone),
			Name:         StageName(StageInvokeDone),
			Labels:       []string{},
			HelpTemplate: "wait invoke down %s",
			Buckets:      []float64{500, 1000, 3000, 5000},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageStopRecvLog),
			Name:         StageName(StageStopRecvLog),
			Labels:       []string{},
			HelpTemplate: "stop receive log %s",
			Buckets:      []float64{1, 2, 5, 10, 30},
			HasSummary:   true,
		},
		{
			MetricType:   metric.MetricTypeHistogram,
			Index:        StageName(StageCleanup),
			Name:         StageName(StageCleanup),
			Labels:       []string{},
			HelpTemplate: "clean up %s",
			Buckets:      []float64{500, 1000, 3000, 5000},
			HasSummary:   true,
		},
	}
)

type rtCtrlInvokeStage = int

const (
	StageWaitRuntime rtCtrlInvokeStage = iota
	StageStartRecvLog
	StageInvokeFunc
	StageSendRequest
	StageRecvResponse
	StageInvokeDone
	StageInvokeReportDone
	StageStopRecvLog
	StageCleanup
)

var (
	stageName = [StageCleanup + 1]string{
		"wait_runtime",
		"start_recvlog",
		"invoke_func",
		"send_request",
		"recv_response",
		"invoke_done",
		"invoke_report_done",
		"stop_recvlog",
		"clean_up",
	}
)

func StageName(stage rtCtrlInvokeStage) string {
	return stageName[stage]
}

type metricStage struct {
	stage    rtCtrlInvokeStage
	duration time.Duration
}

func (i *metricStage) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddFloat64("cost_ms", float64(i.duration.Nanoseconds())/float64(time.Millisecond))
	return nil
}

type RtCtrlInvokeMetric struct {
	requestID    string
	startT       time.Time
	startStage   time.Time
	metricStages []*metricStage
}

func NewRtCtrlInvokeMetric(requestID string) *RtCtrlInvokeMetric {
	now := time.Now()

	return &RtCtrlInvokeMetric{
		requestID:    requestID,
		startT:       now,
		startStage:   now,
		metricStages: make([]*metricStage, 0, StageCleanup+1),
	}
}

func (i *RtCtrlInvokeMetric) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, item := range i.metricStages {
		enc.AddObject(StageName(item.stage), item)
	}
	return nil
}

func InitRtCtrlMetric() error {
	return metric.Register("rtInvoke", stages)
}

func (r *RtCtrlInvokeMetric) StepDone(stage rtCtrlInvokeStage) {
	now := time.Now()
	r.metricStages = append(r.metricStages, &metricStage{
		stage:    stage,
		duration: now.Sub(r.startStage),
	})

	r.startStage = now
}

func (r *RtCtrlInvokeMetric) ObserveAll() {
	for _, m := range r.metricStages {
		metric.Observe(StageName(m.stage), float64(m.duration/time.Microsecond))
	}
}
