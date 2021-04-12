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

package runc

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/baidu/easyfaas/pkg/funclet/runtime/api"
	"github.com/baidu/easyfaas/pkg/util/json"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

type RuncCtl interface {
	Create(c *CreateOpts) error
	UpdateResource(u *UpdateResourceOpts) (err error)
	Start(id string) error
	Run(c *CreateOpts) error
	RunWithStdio(c *CreateOpts, stdout io.WriteCloser, stderr io.WriteCloser) error
	Delete(id string, force bool) error
	State(id string) (container *api.Container, err error)
	List() (list []*api.Container, err error)
	Kill(id string, signal string, all bool) (err error)
	Pause(ID string) (err error)
	Resume(ID string) (err error)
}

type RuncConfig struct {
	Command string
	Logger  *logs.Logger
}

const (
	DefaultCommand = "runc"
)

func (r *RuncConfig) command(args ...string) error {
	command := r.Command
	if command == "" {
		command = DefaultCommand
	}
	cmd := exec.Command(command, args...)
	logs.V(9).Infof("start command %v", cmd.Args)
	if err := cmd.Start(); err != nil {
		if exErr, ok := err.(*exec.Error); ok {
			if exErr.Err == exec.ErrNotFound || exErr.Err == os.ErrNotExist {
				return fmt.Errorf("runc not installed on system")
			}
		}
		return err
	}

	if err := cmd.Wait(); err != nil {
		if sysErr, ok := err.(*os.SyscallError); ok {
			// ignore error "no child process"
			if e := sysErr.Err.(syscall.Errno); e == syscall.ECHILD {
				return nil
			}
		} else {
			logs.Errorf("command [%s] cmd.Wait err %s", cmd.Args, err)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				logs.Errorf("exit code %+v %+v", ws, exitErr)
			}
			return fmt.Errorf("runc execution error")
		}
		return err
	}
	return nil
}

func (r *RuncConfig) commandOutput(args ...string) (data []byte, err error) {
	command := r.Command
	if command == "" {
		command = DefaultCommand
	}
	cmd := exec.Command(command, args...)
	logs.V(9).Infof("start command %v", cmd.Args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		if exErr, ok := err.(*exec.Error); ok {
			if exErr.Err == exec.ErrNotFound || exErr.Err == os.ErrNotExist {
				return nil, fmt.Errorf("runc not installed on system")
			}
		}
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		if sysErr, ok := err.(*os.SyscallError); ok {
			// ignore error "no child process"
			if e := sysErr.Err.(syscall.Errno); e == syscall.ECHILD {
				return stdout.Bytes(), nil
			}
		} else {
			logs.Errorf("command [%s] cmd.Wait err %s", cmd.Args, err)
		}
		logs.V(9).Infof("command %s stdout %s stderr %s", cmd.Args, stdout.Bytes(), stderr.Bytes())
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				logs.Errorf("exit code %+v %+v", ws, exitErr)
			}
			return stderr.Bytes(), fmt.Errorf("runc execution error")
		}
		return stderr.Bytes(), err
	}
	return stdout.Bytes(), nil
}

type CreateOpts struct {
	ID      string
	Bundle  string
	PidFile string
	Detach  bool
}

func (o *CreateOpts) args() (out []string, err error) {
	if o.Bundle != "" {
		abs, err := filepath.Abs(o.Bundle)
		if err != nil {
			return nil, err
		}
		out = append(out, "--bundle", abs)
	}
	if o.PidFile != "" {
		abs, err := filepath.Abs(o.PidFile)
		if err != nil {
			return nil, err
		}
		out = append(out, "--pid-file", abs)
	}
	if o.Detach {
		out = append(out, "--detach")
	}
	return out, nil
}

/*
USAGE:
   runc run [command options] <container-id>

OPTIONS:
   --bundle value, -b value  path to the root of the bundle directory, defaults to the current directory
   --console-socket value    path to an AF_UNIX socket which will receive a file descriptor referencing the master end of the console's pseudoterminal
   --detach, -d              detach from the container's process
   --pid-file value          specify the file to write the process id to
   --no-subreaper            disable the use of the subreaper used to reap reparented processes
   --no-pivot                do not use pivot root to jail process inside rootfs.  This should be used whenever the rootfs is on top of a ramdisk
   --no-new-keyring          do not create a new session keyring for the container.  This will cause the container to inherit the calling processes session key
   --preserve-fds value      Pass N additional file descriptors to the container (stdio + $LISTEN_FDS + N in total) (default: 0)

*/
func (r *RuncConfig) Run(c *CreateOpts) error {
	oargs, err := c.args()
	if err != nil {
		return err
	}
	args := append([]string{"run"}, oargs...)
	args = append(args, c.ID)
	err = r.command(args...)
	if err != nil {
		return err
	}
	return nil
}

func (r *RuncConfig) RunWithStdio(c *CreateOpts, stdout io.WriteCloser, stderr io.WriteCloser) error {
	oargs, err := c.args()
	if err != nil {
		return err
	}
	args := append([]string{"run"}, oargs...)
	args = append(args, c.ID)
	cmd := exec.Command(DefaultCommand, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	logs.V(9).Infof("start command %v", cmd.Args)
	if err := cmd.Start(); err != nil {
		if exErr, ok := err.(*exec.Error); ok {
			if exErr.Err == exec.ErrNotFound || exErr.Err == os.ErrNotExist {
				return fmt.Errorf("runc not installed on system")
			}
		}
		return err
	}
	stdout.Close()
	stderr.Close()
	if err := cmd.Wait(); err != nil {
		if sysErr, ok := err.(*os.SyscallError); ok {
			// ignore error "no child process"
			if e := sysErr.Err.(syscall.Errno); e == syscall.ECHILD {
				return nil
			}
		} else {
			logs.Errorf("command [%s] cmd.Wait err %s", cmd.Args, err)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				logs.Errorf("exit code %+v %+v", ws, exitErr)
			}
			return fmt.Errorf("runc execution error")
		}
		return err
	}
	return nil
}

/*
USAGE:
   runc create [command options] <container-id>
OPTIONS:
   --bundle value, -b value  path to the root of the bundle directory, defaults to the current directory
   --console-socket value    path to an AF_UNIX socket which will receive a file descriptor referencing the master end of the console's pseudoterminal
   --pid-file value          specify the file to write the process id to
   --no-pivot                do not use pivot root to jail process inside rootfs.  This should be used whenever the rootfs is on top of a ramdisk
   --no-new-keyring          do not create a new session keyring for the container.  This will cause the container to inherit the calling processes session key
   --preserve-fds value      Pass N additional file descriptors to the container (stdio + $LISTEN_FDS + N in total) (default: 0)
*/
func (r *RuncConfig) Create(c *CreateOpts) error {
	oargs, err := c.args()
	if err != nil {
		return err
	}
	args := append([]string{"create"}, oargs...)
	args = append(args, c.ID)
	err = r.command(args...)
	if err != nil {
		return err
	}
	return nil
}

/*
USAGE:
   runc start <container-id>
*/
func (r *RuncConfig) Start(id string) error {
	args := append([]string{"start"}, id)
	err := r.command(args...)
	if err != nil {
		return err
	}
	return nil
}

/*
USAGE:
   runc delete [command options] <container-id>

OPTIONS:
   --force, -f  Forcibly deletes the container if it is still running (uses SIGKILL)
*/
func (r *RuncConfig) Delete(id string, force bool) error {
	args := append([]string{"delete"}, id)
	if force {
		args = append(args, "--force")
	}
	err := r.command(args...)
	if err != nil {
		return err
	}
	return nil
}

/*
USAGE:
   runc state <container-id>
*/
func (r *RuncConfig) State(id string) (container *api.Container, err error) {
	args := append([]string{"state"}, id)
	data, err := r.commandOutput(args...)
	if err != nil {
		return nil, fmt.Errorf("runc state err %s", string(data))
	}
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, err
	}
	return container, nil
}

/*
USAGE:
   runc list [command options]

OPTIONS:
   --format value, -f value  select one of: table or json (default: "table")
   --quiet, -q               display only container IDs
*/
func (r *RuncConfig) List() (list []*api.Container, err error) {
	args := append([]string{"list"},
		"--format=json")
	data, err := r.commandOutput(args...)
	if err != nil {
		logs.Errorf("runc list failed: err %+v", err)
	}
	list = make([]*api.Container, 0)
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

/*
USAGE:
   runc kill [command options] <container-id> [signal]

OPTIONS:
   --all, -a  send the specified signal to all processes inside the container
*/
func (r *RuncConfig) Kill(id string, signal string, all bool) (err error) {
	args := append([]string{"kill"}, id)
	args = append(args, signal)
	if all {
		args = append(args, "--all")
	}
	err = r.command(args...)
	if err != nil {
		return err
	}
	return nil
}

type UpdateResourceOpts struct {
	ID       string
	Memory   int64
	CPUQuota int64
}

func (o *UpdateResourceOpts) args() (out []string, err error) {
	if o.Memory != 0 {
		out = append(out, "--memory", strconv.FormatInt(o.Memory, 10))
	}
	if o.CPUQuota != 0 {
		out = append(out, "--cpu-quota", strconv.FormatInt(o.CPUQuota, 10))
	}
	return out, nil
}

/*
USAGE:
   runc update [command options] <container-id>

OPTIONS:
   // ...
   --blkio-weight value        Specifies per cgroup weight, range is from 10 to 1000 (default: 0)
   --cpu-period value          CPU CFS period to be used for hardcapping (in usecs). 0 to use system default
   --cpu-quota value           CPU CFS hardcap limit (in usecs). Allowed cpu time in a given period
   --cpu-share value           CPU shares (relative weight vs. other containers)
   --cpu-rt-period value       CPU realtime period to be used for hardcapping (in usecs). 0 to use system default
   --cpu-rt-runtime value      CPU realtime hardcap limit (in usecs). Allowed cpu time in a given period
   --cpuset-cpus value         CPU(s) to use
   --cpuset-mems value         Memory node(s) to use
   --kernel-memory value       Kernel memory limit (in bytes)
   --kernel-memory-tcp value   Kernel memory limit (in bytes) for tcp buffer
   --memory value              Memory limit (in bytes)
   --memory-reservation value  Memory reservation or soft_limit (in bytes)
   --memory-swap value         Total memory usage (memory + swap); set '-1' to enable unlimited swap
   --pids-limit value          Maximum number of pids allowed in the container (default: 0)
   --l3-cache-schema value     The string of Intel RDT/CAT L3 cache schema
   --mem-bw-schema value       The string of Intel RDT/MBA memory bandwidth schema
*/

func (r *RuncConfig) UpdateResource(u *UpdateResourceOpts) (err error) {
	oargs, err := u.args()
	if err != nil {
		return err
	}
	args := append([]string{"update"}, oargs...)
	args = append(args, u.ID)
	err = r.command(args...)
	if err != nil {
		return err
	}
	return nil
}

/*
USAGE:
   runc resume <container-id>
*/
func (r *RuncConfig) Resume(ID string) (err error) {
	args := append([]string{"resume"}, ID)
	err = r.command(args...)
	if err != nil {
		return err
	}
	return nil
}

/*
USAGE:
   runc pause <container-id>
*/
func (r *RuncConfig) Pause(ID string) (err error) {
	args := append([]string{"pause"}, ID)
	err = r.command(args...)
	if err != nil {
		return err
	}
	return nil
}
