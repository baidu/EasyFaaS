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

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	"github.com/baidu/easyfaas/cmd/funclet/app"
	"github.com/baidu/easyfaas/cmd/funclet/options"
	"github.com/baidu/easyfaas/pkg/util/flag"
	"github.com/baidu/easyfaas/pkg/util/logs"
	"github.com/baidu/easyfaas/pkg/version/verflag"
)

func main() {
	s := options.NewOptions()
	s.AddFlags(pflag.CommandLine)

	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	verflag.PrintAndExitIfRequested()
	stopCh, finishCh := SetupSignalHandler()
	if err := app.Run(s, stopCh, finishCh); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func SetupSignalHandler() (<-chan struct{}, chan struct{}) {
	stop := make(chan struct{})
	finishCh := make(chan struct{})
	sigc := make(chan os.Signal, 2048)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGCHLD)
	var exit bool
	go func() {
		for {
			select {
			case <-finishCh:
				exit = true
			}
		}
	}()
	go func() {
		for {
			s := <-sigc
			logs.Debugf("got signal %s\n", s.String())
			switch s {
			case syscall.SIGCHLD:
				for {
					var ws syscall.WaitStatus
					var rus syscall.Rusage
					pid, err := syscall.Wait4(-1, &ws, syscall.WNOHANG, &rus)
					if err != nil {
						if err == syscall.EINTR {
							continue
						}
						if err != syscall.ECHILD {
							logs.Errorf("wait process err %s ", err)
						}
						break
					}
					if pid <= 0 {
						break
					}
					logs.V(6).Infof("child pid %d exits\n", pid)
				}
			case syscall.SIGINT, syscall.SIGTERM:
				if !exit {
					close(stop)
					// grace shutdown timeout with 10 min
					timer := time.NewTimer(10 * time.Minute)
					go func() {
						select {
						case <-timer.C:
							exit = true
						}
					}()
				} else {
					os.Exit(1)
				}
			}
		}
	}()

	return stop, finishCh
}
