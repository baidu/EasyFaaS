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

// Package rtctrl
package rtctrl

import "github.com/spf13/pflag"

const (
	defaultRuntimeServerAddress = "unix:///var/run/faas/.status_v2.sock"
	defaultRunnerServerAddress  = "unix:///var/run/faas/.funclet_v2.sock"
	defaultUserLogFilePath      = "/tmp/userlog"
	defaultUserLogType          = string(UserLogTypePlain)
)

type DispatcherV2Options struct {
	RuntimeServerAddress string
	RunnerServerAddress  string
	UserLogFileDir       string
	UserLogType          string
}

func NewDispatcherV2Options() *DispatcherV2Options {
	return &DispatcherV2Options{
		RuntimeServerAddress: defaultRuntimeServerAddress,
		RunnerServerAddress:  defaultRunnerServerAddress,
		UserLogFileDir:       defaultUserLogFilePath,
		UserLogType:          defaultUserLogType,
	}
}
func (s *DispatcherV2Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.RuntimeServerAddress, "dispatcher-address",
		s.RuntimeServerAddress, "runtime server address (log and invoke)")
	fs.StringVar(&s.RunnerServerAddress, "runner-dispatcher-address",
		s.RunnerServerAddress, "runner server address")
	fs.StringVar(&s.UserLogFileDir, "userlog-filepath",
		s.UserLogFileDir, "user log storage path")
	fs.StringVar(&s.UserLogType, "userlog-type",
		s.UserLogType, "user log type (eg. plain,json)")
}

type RuntimeConfigOptions struct {
	// Time to wait for runtime to connect (Cold Start)
	// Units: seconds
	WaitRuntimeAliveTimeout int
}

func NewRuntimeConfigOptions() *RuntimeConfigOptions {
	return &RuntimeConfigOptions{
		WaitRuntimeAliveTimeout: 3,
	}
}
func (s *RuntimeConfigOptions) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&s.WaitRuntimeAliveTimeout, "runtime-alive-timeout",
		s.WaitRuntimeAliveTimeout, "Timeout(s) to wait runtime alive")
}
