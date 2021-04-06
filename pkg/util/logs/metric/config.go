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

import "fmt"

type metricType string

const (
	MetricTypeCounter   metricType = "counter"
	MetricTypeGauge     metricType = "gauge"
	MetricTypeHistogram metricType = "histogram"
)

type MetricConfig struct {
	Index        string
	Name         string
	Labels       []string
	HelpTemplate string
	Buckets      []float64
	MetricType   metricType
	HasSummary   bool
}

func (m *MetricConfig) Validator() error {
	if len(m.Index) == 0 {
		return fmt.Errorf("index is required")
	}

	if len(m.Name) == 0 {
		return fmt.Errorf("name is required")
	}

	switch m.MetricType {
	case MetricTypeCounter, MetricTypeGauge:
		if m.HasSummary {
			return fmt.Errorf("metricConfig can not set metricType %s and HasSummary in the same time", m.MetricType)
		}
	}

	return nil
}
