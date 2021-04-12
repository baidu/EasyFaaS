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
	"runtime"

	"github.com/spf13/pflag"

	"github.com/baidu/easyfaas/cmd/controller/app"
	"github.com/baidu/easyfaas/cmd/controller/options"
	"github.com/baidu/easyfaas/pkg/util/flag"
	"github.com/baidu/easyfaas/pkg/util/logs"
	"github.com/baidu/easyfaas/pkg/version/verflag"
)

func main() {
	s := options.NewOptions()
	s.AddFlags(pflag.CommandLine)

	flag.InitFlags()
	logs.InitLogs()
	logs.InitSummaryLogs()
	defer logs.FlushLogs()

	verflag.PrintAndExitIfRequested()
	logs.Infof("origin go process num %d", runtime.GOMAXPROCS(0))
	runtime.GOMAXPROCS(s.GoMaxProcs)
	logs.Infof("set go process num %d", runtime.GOMAXPROCS(0))
	if err := app.Run(s); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
