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

// Package command
package command

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/baidu/openless/pkg/util/logs"
)

func CommandOutput(command string, args ...string) (data []byte, err error) {
	cmd := exec.Command(command, args...)
	logs.V(9).Infof("start command %v", cmd.Args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		if exErr, ok := err.(*exec.Error); ok {
			if exErr.Err == exec.ErrNotFound || exErr.Err == os.ErrNotExist {
				return nil, fmt.Errorf("command %s not installed on system", command)
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
			return stderr.Bytes(), fmt.Errorf("command %s execution error", command)
		}
		return stderr.Bytes(), err
	}
	return stdout.Bytes(), nil
}
