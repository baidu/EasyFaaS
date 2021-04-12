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

package options

import (
	"github.com/spf13/pflag"

	"github.com/baidu/easyfaas/pkg/server"
)

type FeatureOptions struct {
	EnableProfiling           bool
	EnableContentionProfiling bool
	EnableMetrics             bool
	SummaryOverheadMs         int
	// EnableSwaggerUI           bool
}

func NewFeatureOptions() *FeatureOptions {
	defaults := server.NewConfig()

	return &FeatureOptions{
		EnableProfiling:           defaults.EnableProfiling,
		EnableContentionProfiling: defaults.EnableContentionProfiling,
		EnableMetrics:             defaults.EnableMetrics,
		SummaryOverheadMs:         defaults.SummaryOverheadMs,
		// EnableSwaggerUI:           defaults.EnableSwaggerUI,
	}
}

func (o *FeatureOptions) AddFlags(fs *pflag.FlagSet) {
	if o == nil {
		return
	}

	fs.BoolVar(&o.EnableProfiling, "profiling", o.EnableProfiling,
		"Enable profiling via web interface host:port/debug/pprof/")
	fs.BoolVar(&o.EnableContentionProfiling, "contention-profiling", o.EnableContentionProfiling,
		"Enable lock contention profiling, if profiling is enabled")
	fs.BoolVar(&o.EnableMetrics, "enable-metrics", o.EnableMetrics,
		"Enable prometheus metrics via web interface host:port/metrics")
	fs.IntVar(&o.SummaryOverheadMs, "summary-overhead", o.SummaryOverheadMs,
		"Record request summary log if request overhead longer than summary-overhead ms, set 0 if you want to record all request summary log")
	// fs.BoolVar(&o.EnableSwaggerUI, "enable-swagger-ui", o.EnableSwaggerUI,
	//	"Enables swagger ui on the apiserver at /swagger-ui")
}

func (o *FeatureOptions) ApplyTo(c *server.Config) error {
	if o == nil {
		return nil
	}

	c.EnableProfiling = o.EnableProfiling
	c.EnableContentionProfiling = o.EnableContentionProfiling
	c.EnableMetrics = o.EnableMetrics
	c.SummaryOverheadMs = o.SummaryOverheadMs
	// c.EnableSwaggerUI = o.EnableSwaggerUI

	return nil
}

func (o *FeatureOptions) Validate() []error {
	if o == nil {
		return nil
	}

	errs := []error{}
	return errs
}
