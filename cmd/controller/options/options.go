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

// Package options
package options

import (
	"runtime"

	"github.com/spf13/pflag"

	"github.com/baidu/openless/pkg/funclet/client"
	"github.com/baidu/openless/pkg/controller/function"
	"github.com/baidu/openless/pkg/controller/registry"
	"github.com/baidu/openless/pkg/controller/rtctrl"
	genericoptions "github.com/baidu/openless/pkg/server/options"
)

// ControllerOptions: options of the controller app
type ControllerOptions struct {
	RecommendedOptions           *genericoptions.RecommendedOptions
	FuncletClientOptions         *client.FuncletClientOptions
	DispatcherV2Options          *rtctrl.DispatcherV2Options
	RepositoryOptions            *registry.Options
	HttpTriggerRepositoryOptions *registry.Options
	RuntimeConfigOptions         *rtctrl.RuntimeConfigOptions
	AliasCacheOptions            *function.StorageCacheOptions
	// Task cycle interval
	// Units: seconds
	TaskInterval int

	// Task cycle interval
	// Units: seconds
	MetricsTaskInterval int

	// Runtime maximum idle time
	// Units: seconds
	MaxRuntimeIdle int

	// The longest expire time after runner disconnect
	// Units: seconds
	MaxRunnerDefunct int

	// The longest expire time after runner last reset
	// Units: seconds
	MaxRunnerResetTimeout int

	// Runtime concurrent mode switch
	ConcurrentMode bool
	// HTTP trigger feature switch
	HTTPEnhanced bool

	// Setting the proper go process for the best performance
	GoMaxProcs int

	EnableCanary bool

	SimpleAuth bool
}

func NewOptions() *ControllerOptions {
	return &ControllerOptions{
		RecommendedOptions:           genericoptions.NewRecommendedOptions(),
		FuncletClientOptions:         client.NewFuncletClientOptions(),
		DispatcherV2Options:          rtctrl.NewDispatcherV2Options(),
		RepositoryOptions:            registry.NewOption(),
		HttpTriggerRepositoryOptions: registry.NewEmptyOption(),
		RuntimeConfigOptions:         rtctrl.NewRuntimeConfigOptions(),
		AliasCacheOptions:            function.NewStorageCacheOptions(),
		TaskInterval:                 5,
		MetricsTaskInterval:          10,
		MaxRuntimeIdle:               60,
		MaxRunnerDefunct:             90,
		MaxRunnerResetTimeout:        60,
		ConcurrentMode:               true,
		GoMaxProcs:                   runtime.NumCPU(),
		HTTPEnhanced:                 false,
		EnableCanary:                 false,
		SimpleAuth:                   true,
	}
}

func (s *ControllerOptions) AddFlags(fs *pflag.FlagSet) {
	s.RecommendedOptions.AddFlags(fs)
	s.FuncletClientOptions.AddFuncletClientFlags(fs)
	s.RepositoryOptions.AddFlags("", fs)
	s.HttpTriggerRepositoryOptions.AddFlags("httptrigger", fs)
	s.DispatcherV2Options.AddFlags(fs)
	s.RuntimeConfigOptions.AddFlags(fs)
	s.AliasCacheOptions.AddFlags("alias", fs)
	fs.IntVar(&s.TaskInterval, "task-interval", s.TaskInterval, "cron task interval")
	fs.IntVar(&s.MetricsTaskInterval, "metric-task-interval", s.MetricsTaskInterval, "metric task interval")
	fs.IntVar(&s.MaxRuntimeIdle, "max-runtime-idle", s.MaxRuntimeIdle, "max runtime idle timeout")
	fs.IntVar(&s.MaxRunnerDefunct, "max-runner-defunct", s.MaxRunnerDefunct, "max runner defunct timeout")
	fs.IntVar(&s.MaxRunnerResetTimeout, "max-runner-reset-timeout", s.MaxRunnerResetTimeout, "max runner reset timeout")
	fs.BoolVar(&s.ConcurrentMode, "concurrent-mode", s.ConcurrentMode, "whether runtime run concurrently")
	fs.IntVar(&s.GoMaxProcs, "maxprocs", s.GoMaxProcs, "go max procs")
	fs.BoolVar(&s.HTTPEnhanced, "http-enhanced", s.HTTPEnhanced, "whether to equip with http trigger feature")
	fs.BoolVar(&s.SimpleAuth, "enable-simple-auth", s.SimpleAuth, "whether to use simple auth")
	fs.BoolVar(&s.EnableCanary, "enable-canary", s.EnableCanary, "whether to enable canary")
}
