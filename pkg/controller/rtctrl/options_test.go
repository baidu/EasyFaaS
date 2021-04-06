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

import (
	"reflect"
	"testing"

	"github.com/spf13/pflag"
)

func TestDispatcherV2Options_AddFlags(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	disOpt := NewDispatcherV2Options()
	disOpt.AddFlags(fs)
	args := []string{
		"--dispatcher-address=/tmp/.test.sock",
		"--runner-dispatcher-address=/tmp/.runner-test.sock",
	}
	fs.Parse(args)
	expected := &DispatcherV2Options{
		RunnerServerAddress:  "/tmp/.runner-test.sock",
		RuntimeServerAddress: "/tmp/.test.sock",
		UserLogFileDir:       defaultUserLogFilePath,
		UserLogType:          defaultUserLogType,
	}
	if !reflect.DeepEqual(expected, disOpt) {
		t.Errorf("Got different run options than expected.")
	}
}

func TestRuntimeConfigOptions_AddFlags(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	disOpt := NewRuntimeConfigOptions()
	disOpt.AddFlags(fs)
	args := []string{
		"--runtime-alive-timeout=10",
	}
	fs.Parse(args)
	expected := &RuntimeConfigOptions{
		WaitRuntimeAliveTimeout: 10,
	}
	if !reflect.DeepEqual(expected, disOpt) {
		t.Errorf("Got different run options than expected.")
	}
}
