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

// Package runner
package runner

import (
	"fmt"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	api2 "github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/funclet/runtime/api"
)

const (
	SpecConfig       = "config.json"
	DefaultDNSConfig = "/etc/resolv.conf"
	RUNTIMEUID       = "CFC_RUNTIME_UID=1001"
	RUNTIMEGID       = "CFC_RUNTIME_GID=1001"
)

type RunnerManagerInterface interface {
	GenerateRunnerConfig(c *RunnerConfig) *RunnerSpec
}

type Manager struct {
	Mode          string
	Options       *RunnerSpecOption
	defaultConfig *api.ResourceConfig
}

func NewRunnerManager(o *RunnerSpecOption, config *api.ResourceConfig, mode string) *Manager {
	return &Manager{
		Mode:          mode,
		Options:       o,
		defaultConfig: config,
	}
}

func (m *Manager) runnerExample() *RunnerSpec {
	return &specs.Spec{
		Version: specs.Version,
		Root: &specs.Root{
			Path:     m.Options.RootfsPath,
			Readonly: true,
		},
		Process: &specs.Process{
			Terminal: false,
			User:     specs.User{},
			Args:     []string{},
			Env: []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"TERM=xterm",
			},
			Cwd:             "/",
			NoNewPrivileges: true,
			Capabilities: &specs.LinuxCapabilities{
				Bounding: []string{
					"CAP_AUDIT_WRITE",
					"CAP_KILL",
					"CAP_NET_BIND_SERVICE",
					"CAP_SETUID",
					"CAP_SETGID",
				},
				Permitted: []string{
					"CAP_AUDIT_WRITE",
					"CAP_KILL",
					"CAP_NET_BIND_SERVICE",
					"CAP_SETUID",
					"CAP_SETGID",
				},
				Inheritable: []string{
					"CAP_AUDIT_WRITE",
					"CAP_KILL",
					"CAP_NET_BIND_SERVICE",
					"CAP_SETUID",
					"CAP_SETGID",
				},
				Ambient: []string{
					"CAP_AUDIT_WRITE",
					"CAP_KILL",
					"CAP_NET_BIND_SERVICE",
					"CAP_SETUID",
					"CAP_SETGID",
				},
				Effective: []string{
					"CAP_AUDIT_WRITE",
					"CAP_KILL",
					"CAP_NET_BIND_SERVICE",
					"CAP_SETUID",
					"CAP_SETGID",
				},
			},
			Rlimits: []specs.POSIXRlimit{
				{
					Type: "RLIMIT_NOFILE",
					Hard: uint64(1024 * 1024),
					Soft: uint64(1024 * 1024),
				},
			},
		},
		Hostname: "runner",
		Mounts: []specs.Mount{
			{
				Destination: "/proc",
				Type:        "proc",
				Source:      "proc",
				Options:     nil,
			},
			{
				Destination: "/dev",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"nosuid", "strictatime", "mode=755", "size=65536k"},
			},
			{
				Destination: "/dev/pts",
				Type:        "devpts",
				Source:      "devpts",
				Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620", "gid=5"},
			},
			{
				Destination: "/dev/shm",
				Type:        "tmpfs",
				Source:      "shm",
				Options:     []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"},
			},
			{
				Destination: "/dev/mqueue",
				Type:        "mqueue",
				Source:      "mqueue",
				Options:     []string{"nosuid", "noexec", "nodev"},
			},
			{
				Destination: "/sys",
				Type:        "sysfs",
				Source:      "sysfs",
				Options:     []string{"nosuid", "noexec", "nodev", "ro"},
			},
			{
				Destination: "/sys/fs/cgroup",
				Type:        "cgroup",
				Source:      "cgroup",
				Options:     []string{"nosuid", "noexec", "nodev", "relatime", "ro"},
			},
		},
		Linux: &specs.Linux{
			MaskedPaths: []string{
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug",
				"/sys/firmware",
				"/proc/scsi",
			},
			ReadonlyPaths: []string{
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
			Resources: &specs.LinuxResources{
				Devices: []specs.LinuxDeviceCgroup{
					{
						Allow:  false,
						Access: "rwm",
					},
				},
			},
			Namespaces: []specs.LinuxNamespace{
				{
					Type: "pid",
				},
				{
					Type: "network",
				},
				{
					Type: "ipc",
				},
				{
					Type: "uts",
				},
				{
					Type: "mount",
				},
			},
		},
	}
}

func (m *Manager) GenerateRunnerConfig(c *RunnerConfig) *RunnerSpec {
	if m.Mode == api2.RunningModeIDE {
		return m.getIDERunnerConfig(c)
	}
	return m.getCommonRunnerConfig(c)
}

func (m *Manager) getCommonRunnerConfig(c *RunnerConfig) *RunnerSpec {
	defaultSpec := m.runnerExample()
	// set uid gid
	defaultSpec.Process.Env = append(defaultSpec.Process.Env, RUNTIMEUID)
	defaultSpec.Process.Env = append(defaultSpec.Process.Env, RUNTIMEGID)
	defaultSpec.Process.Args = strings.Split(m.Options.RunnerCmd, " ")
	// hostname
	defaultSpec.Hostname = c.HostName
	// invoker sock
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: m.Options.TargetInvokerSocketPath,
		Type:        "bind",
		Source:      m.Options.InvokerSocketPath,
		Options:     []string{"rbind", "ro"},
	})
	// runtime sock
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: m.Options.TargetRuntimeSocketPath,
		Type:        "bind",
		Source:      c.RuntimeSocketPath,
		Options:     []string{"nosuid", "rbind", "rw", "mode=777"},
	})
	// etc hosts
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: m.Options.TargetHostsPath,
		Type:        "bind",
		Source:      c.HostsPath,
		Options:     []string{"rbind", "ro"},
	})
	// defaultConfig
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: m.Options.TargetConfPath,
		Type:        "bind",
		Source:      c.ConfigPath,
		Options:     []string{"rbind", "ro", "slave"},
	})
	// code
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: m.Options.TargetCodePath,
		Type:        "bind",
		Source:      c.CodePath,
		Options:     []string{"rbind", "ro", "slave"},
	})
	// runtime
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: m.Options.TargetRuntimePath,
		Type:        "bind",
		Source:      c.RuntimePath,
		Options:     []string{"rbind", "ro", "slave"},
	})
	// tmp
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: "/tmp",
		Type:        "bind",
		Source:      c.TmpPath,
		Options:     []string{"rbind", "rw"},
	})
	// dns
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: DefaultDNSConfig,
		Type:        "bind",
		Source:      DefaultDNSConfig,
		Options:     []string{"rbind", "ro"},
	})

	// resources
	if c.ResourceConfig == nil {
		c.ResourceConfig = m.defaultConfig
	}
	defaultSpec.Linux.Resources.Memory = &specs.LinuxMemory{
		Limit: c.ResourceConfig.Memory,
	}
	defaultSpec.Linux.Resources.CPU = &specs.LinuxCPU{
		Shares: c.ResourceConfig.CpuShares,
		Quota:  c.ResourceConfig.CpuQuota,
		Period: c.ResourceConfig.CpuPeriod,
	}
	return defaultSpec
}

func (m *Manager) getIDERunnerConfig(c *RunnerConfig) *RunnerSpec {
	defaultSpec := m.runnerExample()
	// hostname
	defaultSpec.Hostname = c.HostName
	defaultSpec.Process.User.UID = 1000
	defaultSpec.Process.User.GID = 1000
	defaultSpec.Process.Args = strings.Split(m.Options.RunnerCmd, " ")
	defaultSpec.Process.Env = append(defaultSpec.Process.Env, "LC_ALL=C.UTF-8")
	defaultSpec.Process.Env = append(defaultSpec.Process.Env, "LANG=C.UTF-8")
	defaultSpec.Process.Env = append(defaultSpec.Process.Env,
		fmt.Sprintf("CFC_VSCODE_SERVE_SOCKET=%s/.vscode.sock", m.Options.TargetRuntimeSocketPath))
	defaultSpec.Process.Env = append(defaultSpec.Process.Env,
		fmt.Sprintf("CFC_MANAGER_SOCKET=%s", m.Options.VscodeManagerSocketPath))

	// invoker sock
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: m.Options.TargetInvokerSocketPath,
		Type:        "bind",
		Source:      m.Options.InvokerSocketPath,
		Options:     []string{"rbind", "ro"},
	})
	// runtime sock
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: m.Options.TargetRuntimeSocketPath,
		Type:        "bind",
		Source:      c.RuntimeSocketPath,
		Options:     []string{"nosuid", "bind", "rw", "mode=777"},
	})
	// etc hosts
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: m.Options.TargetHostsPath,
		Type:        "bind",
		Source:      c.HostsPath,
		Options:     []string{"rbind", "ro"},
	})
	// code
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: m.Options.TargetCodePath,
		Type:        "bind",
		Source:      c.CodePath,
		Options:     []string{"uid=1000", "gid=1000", "bind", "rw", "mode=777", "shared"},
	})
	// runtime
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: m.Options.TargetRuntimePath,
		Type:        "bind",
		Source:      c.RuntimePath,
		Options:     []string{"rbind", "ro", "slave"},
	})
	//vscode
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: "/home",
		Type:        "tmpfs",
		Source:      "tmpfs",
		Options:     []string{"uid=1000", "gid=1000", "strictatime", "rw", "mode=777"},
	})
	//vscode
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: "/.vscode-remote",
		Type:        "tmpfs",
		Source:      "tmpfs",
		Options:     []string{"uid=1000", "gid=1000", "strictatime", "rw", "mode=777"},
	})
	// tmp
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: "/tmp",
		Type:        "bind",
		Source:      c.TmpPath,
		Options:     []string{"rbind", "rw"},
	})
	// dns
	defaultSpec.Mounts = append(defaultSpec.Mounts, specs.Mount{
		Destination: DefaultDNSConfig,
		Type:        "bind",
		Source:      DefaultDNSConfig,
		Options:     []string{"rbind", "ro"},
	})

	// resources
	if c.ResourceConfig == nil {
		c.ResourceConfig = m.defaultConfig
	}
	defaultSpec.Linux.Resources.Memory = &specs.LinuxMemory{
		Limit: c.ResourceConfig.Memory,
	}
	defaultSpec.Linux.Resources.CPU = &specs.LinuxCPU{
		Shares: c.ResourceConfig.CpuShares,
		Quota:  c.ResourceConfig.CpuQuota,
		Period: c.ResourceConfig.CpuPeriod,
	}
	return defaultSpec
}
