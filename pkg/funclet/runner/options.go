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

package runner

import (
	"github.com/spf13/pflag"
)

type RunnerSpecOption struct {
	RunnerCmd               string
	RootfsPath              string
	EtcPath                 string
	CodePath                string
	ConfPath                string
	RuntimePath             string
	InvokerSocketPath       string
	RuntimeSocketPath       string
	TargetHostsPath         string
	TargetConfPath          string
	TargetCodePath          string
	TargetRuntimePath       string
	TargetTmpPath           string
	TargetInvokerSocketPath string
	TargetRuntimeSocketPath string
	VscodeManagerSocketPath string
}

func NewRunnerSpecOption() *RunnerSpecOption {
	return &RunnerSpecOption{
		RunnerCmd:               "/init",
		RootfsPath:              "/var/faas/runner/rootfs",
		EtcPath:                 "/var/faas/runner-spec/%s/conf",
		CodePath:                "/var/faas/runner-data/%s/task",
		ConfPath:                "/var/faas/runner-data/%s/conf",
		RuntimePath:             "/var/faas/runner-data/%s/runtime",
		InvokerSocketPath:       "/var/run/faas",
		RuntimeSocketPath:       "/var/run/faas/%s",
		TargetHostsPath:         "/etc/hosts",
		TargetCodePath:          "/var/task",
		TargetConfPath:          "/etc/faas",
		TargetRuntimePath:       "/var/runtime",
		TargetTmpPath:           "/tmp",
		TargetInvokerSocketPath: "/var/run/faas",
		TargetRuntimeSocketPath: "/var/run/faas-runtime",
		VscodeManagerSocketPath: "/var/run/faas/.vscode_status.sock",
	}
}
func (s *RunnerSpecOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.RootfsPath, "rootfs-path", s.RootfsPath, "runner rootfs path")
	fs.StringVar(&s.RunnerCmd, "runner-cmd", s.RunnerCmd, "command of runner")
	fs.StringVar(&s.EtcPath, "runner-etc-path", s.EtcPath, "funclet etc path for runner")
	fs.StringVar(&s.CodePath, "runner-code-path", s.CodePath, "funclet code path for runner")
	fs.StringVar(&s.ConfPath, "runner-conf-path", s.ConfPath, "funclet conf path for runner")
	fs.StringVar(&s.RuntimePath, "runner-runtime-path", s.RuntimePath, "funclet runtime path for runner")
	fs.StringVar(&s.InvokerSocketPath, "runner-invoker-socket-path", s.InvokerSocketPath, "invoker socket path for runner")
	fs.StringVar(&s.RuntimeSocketPath, "runner-runtime-socket-path", s.RuntimeSocketPath, "directory path of runtime socket")
	fs.StringVar(&s.TargetHostsPath, "runner-target-hosts-path", s.TargetHostsPath, "runner hosts path")
	fs.StringVar(&s.TargetCodePath, "runner-target-code-path", s.TargetCodePath, "runner code path")
	fs.StringVar(&s.TargetConfPath, "runner-target-conf-path", s.TargetConfPath, "runner conf path")
	fs.StringVar(&s.TargetRuntimePath, "runner-target-runtime-path", s.TargetRuntimePath, "runner runtime path")
	fs.StringVar(&s.TargetTmpPath, "runner-target-tmp-path", s.TargetTmpPath, "runner writable directory")
}
