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

package metric

import (
	"fmt"
	"time"

	"github.com/baidu/openless/pkg/util/logs"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

var (
	metricNamespace = "namespace"
)

var (
	metricHistogramVecMap = make(map[string]*prometheus.HistogramVec)
	metricCounterVecMap   = make(map[string]*prometheus.CounterVec)
	metricGaugeVecMap     = make(map[string]*prometheus.GaugeVec)
	metricSummaryVecMap   = make(map[string]*prometheus.SummaryVec)
)

func InitMetric(namespace string) {
	metricNamespace = namespace
}

func Register(subsystem string, configs []MetricConfig) error {
	// check all config is valid
	for _, c := range configs {
		if err := c.Validator(); err != nil {
			return err
		}
	}

	for _, c := range configs {
		switch c.MetricType {
		case MetricTypeCounter:
			counter := prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: metricNamespace,
				Subsystem: subsystem,
				Name:      c.Name,
				Help:      c.HelpTemplate,
			}, c.Labels)
			prometheus.MustRegister(counter)
			metricCounterVecMap[c.Index] = counter

		case MetricTypeGauge:
			gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: metricNamespace,
				Subsystem: subsystem,
				Name:      c.Name,
				Help:      c.HelpTemplate,
			}, c.Labels)
			prometheus.MustRegister(gauge)
			metricGaugeVecMap[c.Index] = gauge

		case MetricTypeHistogram:
			histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: metricNamespace,
				Subsystem: subsystem,
				Name:      c.Name,
				Help:      c.HelpTemplate,
				Buckets:   c.Buckets,
			}, c.Labels)
			prometheus.MustRegister(histogram)
			metricHistogramVecMap[c.Index] = histogram

			if c.HasSummary {
				summary := prometheus.NewSummaryVec(prometheus.SummaryOpts{
					Namespace: metricNamespace,
					Subsystem: subsystem,
					Name:      fmt.Sprintf("%s_latencies", c.Name),
					Help:      c.HelpTemplate,
					MaxAge:    10 * time.Minute,
				}, c.Labels)
				prometheus.MustRegister(summary)
				metricSummaryVecMap[c.Index] = summary
			}
		default:
		}
	}

	return nil
}

func Observe(index string, value float64, labels ...string) {
	if histogram, ok := metricHistogramVecMap[index]; ok {
		histogram.WithLabelValues(labels...).Observe(value)

		if summary, ok := metricSummaryVecMap[index]; ok {
			summary.WithLabelValues(labels...).Observe(value)
		}
	}
}

func ObserveWithLabels(index string, value float64, labels *prometheus.Labels) {
	if histogram, ok := metricHistogramVecMap[index]; ok {
		histogram.With(*labels).Observe(value)

		if summary, ok := metricSummaryVecMap[index]; ok {
			summary.With(*labels).Observe(value)
		}
	}
}

func GetCounterValue(index string, labels ...string) float64 {
	m := &dto.Metric{}
	if counter, ok := metricCounterVecMap[index]; ok {
		if err := counter.WithLabelValues(labels...).Write(m); err != nil {
			logs.Errorf("write metric failed: %s", err)
			return 0
		}
	}
	return m.Counter.GetValue()
}

func Inc(index string, labels ...string) {
	if counter, ok := metricCounterVecMap[index]; ok {
		counter.WithLabelValues(labels...).Inc()
	}
}

func IncWithLabels(index string, labels *prometheus.Labels) {
	if counter, ok := metricCounterVecMap[index]; ok {
		counter.With(*labels).Inc()
	}
}

func Add(index string, value float64, labels ...string) {
	if counter, ok := metricCounterVecMap[index]; ok {
		counter.WithLabelValues(labels...).Add(value)
	}
}

func AddWithLabels(index string, value float64, labels *prometheus.Labels) {
	if counter, ok := metricCounterVecMap[index]; ok {
		counter.With(*labels).Add(value)
	}
}

func SetGauge(index string, value float64, labels ...string) {
	if gauge, ok := metricGaugeVecMap[index]; ok {
		gauge.WithLabelValues(labels...).Set(value)
	}
}

func SetGaugeWithLabels(index string, value float64, labels *prometheus.Labels) {
	if gauge, ok := metricGaugeVecMap[index]; ok {
		gauge.With(*labels).Set(value)
	}
}

func AddGauge(index string, value float64, labels ...string) {
	if gauge, ok := metricGaugeVecMap[index]; ok {
		gauge.WithLabelValues(labels...).Add(value)
	}
}

func SubGauge(index string, value float64, labels ...string) {
	if gauge, ok := metricGaugeVecMap[index]; ok {
		gauge.WithLabelValues(labels...).Sub(value)
	}
}
