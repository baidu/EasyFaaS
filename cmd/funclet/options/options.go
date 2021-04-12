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

	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/funclet/network"
	"github.com/baidu/easyfaas/pkg/funclet/runner"
	"github.com/baidu/easyfaas/pkg/funclet/runtime"
	"github.com/baidu/easyfaas/pkg/funclet/tmp"
	genericoptions "github.com/baidu/easyfaas/pkg/server/options"
)

type FuncletOptions struct {
	RecommendedOptions    *genericoptions.RecommendedOptions
	ContainerNum          int
	RuntimeCmd            string
	RunnerSpecPath        string
	RunnerDataPath        string
	FuncletApiSocks       string
	ConfPath              string
	RuntimePath           string
	TmpPath               string
	CachePath             string
	InvokerSocks          string
	ListenPath            string
	ListenType            string
	InvokerDispatcherPort int
	KillRuntimeWaitTime   int // times of waiting for process to exit (unit seconds)
	RunningMode           string

	ResourceOption   *runtime.ResourceOption
	RunnerSpecOption *runner.RunnerSpecOption
	NetworkOption    *network.NetworkOption
	TmpStorageOption *tmp.TmpStorageOption
}

func NewOptions() *FuncletOptions {
	return &FuncletOptions{
		RecommendedOptions:    genericoptions.NewRecommendedOptions(),
		ContainerNum:          10,
		RuntimeCmd:            "runc",
		RunnerSpecPath:        "/var/faas/runner-spec",
		RunnerDataPath:        "/var/faas/runner-data",
		FuncletApiSocks:       "/var/run/faas/.funcletapi.sock",
		RuntimePath:           "/var/faas/runtime",
		ConfPath:              "/var/faas/conf",
		CachePath:             "/var/faas/cache",
		TmpPath:               "/var/faas/tmp",
		InvokerSocks:          "/var/run/faas/.server.sock",
		ListenType:            "unix",
		ListenPath:            "/var/run/faas/.funclet.sock",
		InvokerDispatcherPort: 18400,
		KillRuntimeWaitTime:   10,
		RunningMode:           api.RunningModeCommon,
		ResourceOption:        runtime.NewResourceOption(),
		RunnerSpecOption:      runner.NewRunnerSpecOption(),
		NetworkOption:         network.NewNetworkOption(),
		TmpStorageOption:      tmp.NewTmpStorageOption(),
	}
}
func (s *FuncletOptions) AddFlags(fs *pflag.FlagSet) {
	s.RecommendedOptions.AddFlags(fs)
	s.RunnerSpecOption.AddFlags(fs)
	s.NetworkOption.AddFlags(fs)
	s.TmpStorageOption.AddFlags(fs)
	s.ResourceOption.AddFlags(fs)
	fs.IntVar(&s.ContainerNum, "container-num", s.ContainerNum, "num of container")
	fs.StringVar(&s.RuntimeCmd, "runtime-cmd", s.RuntimeCmd, "runtime cli binary path")
	fs.StringVar(&s.RunnerSpecPath, "runner-spec-dir", s.RunnerSpecPath, "runc spec for runner")
	fs.StringVar(&s.RunnerDataPath, "runner-data-dir", s.RunnerDataPath, "runner data")
	fs.StringVar(&s.FuncletApiSocks, "api-sock", s.FuncletApiSocks, "funclet api socks")
	fs.StringVar(&s.ConfPath, "conf-path", s.ConfPath, "Conf path")
	fs.StringVar(&s.TmpPath, "tmp-path", s.TmpPath, "Tmp path")
	fs.StringVar(&s.RuntimePath, "runtime-path", s.RuntimePath, "Runtime path")
	fs.StringVar(&s.CachePath, "cache-path", s.CachePath, "Code cache path")

	fs.StringVar(&s.InvokerSocks, "invoker", s.InvokerSocks, "Unix domain socks of invoker")
	fs.StringVar(&s.ListenPath, "listen", s.ListenPath, "Unix domain socks of funclet")
	fs.StringVar(&s.ListenType, "listentype", s.ListenType, "Type of connect (with runner)")
	fs.IntVar(&s.KillRuntimeWaitTime, "kill-runtime-wait-times", s.KillRuntimeWaitTime, "Times of waiting for process to exit (unit seconds)")
	fs.IntVar(&s.InvokerDispatcherPort, "invoker-dis-port", s.InvokerDispatcherPort, "Port of invoker dispatcher")
	fs.StringVar(&s.RunningMode, "running-mode", s.RunningMode, "Running mode: common,ide; default common")
}
