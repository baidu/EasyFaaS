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
	"reflect"
	"testing"

	"github.com/baidu/openless/pkg/controller/registry"

	"fmt"

	"github.com/spf13/pflag"
)

func TestAddFlags(t *testing.T) {
	s := NewOptions()
	fs := pflag.NewFlagSet("addflagstest", pflag.ContinueOnError)
	s.AddFlags(fs)
	args := []string{
		"--maxprocs=10",
		"--repository-endpoint=http://127.0.0.1:3030",
		"--repository-version=v1/ote",
		"--repository-auth-type=cloud",
		"--repository-auth-params={\"ak\":\"aaa\", \"sk\": \"bbb\"}",
		"--http-enhanced=true",
	}
	fs.Parse(args)
	fmt.Printf("%+v", s)
	expected := NewOptions()
	expected.GoMaxProcs = 10
	expected.RepositoryOptions = &registry.Options{
		Endpoint: "http://127.0.0.1:3030",
		Version:  "v1/ote",
		Auth: &registry.AuthOption{
			Name:     "cloud",
			ParamStr: "{\"ak\":\"aaa\", \"sk\": \"bbb\"}",
		},
	}
	expected.HTTPEnhanced = true
	if !reflect.DeepEqual(expected, s) {
		t.Errorf("Got different run options than expected.")
	}
}
