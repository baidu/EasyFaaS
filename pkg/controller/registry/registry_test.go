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

// Package registry
package registry

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	kunErr "github.com/baidu/easyfaas/pkg/error"

	"github.com/spf13/pflag"

	"github.com/baidu/easyfaas/pkg/auth"
	"github.com/baidu/easyfaas/pkg/util/json"

	"github.com/baidu/easyfaas/pkg/util"

	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/baidu/easyfaas/pkg/api"
)

func TestNewRegistry(t *testing.T) {
	s := NewOption()
	fs := pflag.NewFlagSet("addflagstest", pflag.ContinueOnError)
	s.AddFlags("", fs)
	authType := "wrong"
	args := []string{
		"--repository-version=v1/ote",
		fmt.Sprintf("--repository-auth-type=%s", authType),
		"--repository-auth-params={\"ak\":\"xxx\", \"sk\": \"xxxxxxx\"}",
	}
	fs.Parse(args)
	_, err := NewRegistry(s)
	expectedErr := auth.InvalidAuthSignerError{authType}
	if err != expectedErr {
		t.Errorf("registry expected err %s but got err: %s", expectedErr, err)
		return
	}

	args2 := []string{
		"--repository-version=v1/ote",
		"--repository-auth-type=cloud",
		"--repository-auth-params={\"ak\":\"xxx\", \"sk\": \"xxxxxxx\"",
	}
	fs.Parse(args2)
	_, err = NewRegistry(s)
	if err == nil {
		t.Error("registry expected err, but got nil")
	}
}

func TestRegistryClient_GetFunction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		output := getFunction()
		res, _ := json.Marshal(output)
		w.Write(res)
		return
	}))
	defer ts.Close()

	opt := NewOption()
	opt.Endpoint = ts.URL
	r, err := NewRegistry(opt)
	if err != nil {
		t.Errorf("registry occurred error: %s", err)
		return
	}
	input := api.GetFunctionInput{
		GetFunctionInput: lambda.GetFunctionInput{
			FunctionName: util.String("xxxx"),
		},
		RequestID: "test",
		AccountID: "abc",
	}
	output, err := r.GetFunction(&input)
	if err != nil {
		t.Errorf("get function failed: %s", err)
		return
	}
	t.Logf("function info %v", output)
}

func TestRegistryClient_GetFunctionNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		err := kunErr.NewGenericException(kunErr.BasicError{
			Code:    kunErr.ResourceNotFoundException,
			Cause:   "",
			Message: "The resource specified in the request does not exist",
			Status:  http.StatusNotFound,
		}, nil)
		res, _ := json.Marshal(err)
		w.Write(res)
		return
	}))
	defer ts.Close()

	opt := NewOption()
	opt.Endpoint = ts.URL
	r, err := NewRegistry(opt)
	if err != nil {
		t.Errorf("registry occurred error: %s", err)
		return
	}
	input := api.GetFunctionInput{
		GetFunctionInput: lambda.GetFunctionInput{
			FunctionName: util.String("xxxx"),
		},
		RequestID: "test",
		AccountID: "abc",
	}
	output, err := r.GetFunction(&input)
	if err != nil {
		finalErr := kunErr.GenericKunFinalError(err)
		if finalErr.Status != http.StatusNotFound {
			t.Errorf("get function failed: %s", err)
		}
		return
	}
	t.Logf("function info %v", output)
}

func TestRegistryClient_GetRuntimeConfiguration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		output := getRuntimeConfiguration()
		res, _ := json.Marshal(output)
		w.Write(res)
		return
	}))
	defer ts.Close()

	s := NewOption()
	fs := pflag.NewFlagSet("addflagstest", pflag.ContinueOnError)
	s.AddFlags("", fs)
	args := []string{
		"--repository-version=v1/ote",
		"--repository-auth-type=cloud",
		"--repository-auth-params={\"ak\":\"xxx\", \"sk\": \"xxxxxxx\"}",
	}
	fs.Parse(args)
	s.Endpoint = ts.URL
	r, err := NewRegistry(s)
	if err != nil {
		t.Errorf("registry occurred error: %s", err)
		return
	}
	output, err := r.GetRuntimeConfiguration(&api.GetRuntimeConfigurationInput{RuntimeName: "nodejs12"})
	if err != nil {
		t.Errorf("get runtime failed: %s", err)
		return
	}
	t.Logf("function info %v", output)
}

func getFunction() *api.GetFunctionOutput {
	var timeout int64 = 3
	brn := "brn:faas:function:123:test:$LATEST"
	name := "test"
	uid := "123"
	runtime := "nodejs8.5"

	return &api.GetFunctionOutput{
		Configuration: &api.FunctionConfiguration{
			FunctionConfiguration: lambda.FunctionConfiguration{
				Timeout:      &timeout,
				FunctionArn:  &brn,
				FunctionName: &name,
				Runtime:      &runtime,
			},
			Uid: uid,
		},
		Uid:     uid,
		LogType: "bos",
	}
}

func getRuntimeConfiguration() *api.RuntimeConfiguration {
	return &api.RuntimeConfiguration{
		Name: "nodejs12",
		Bin:  "/bin/bash",
		Path: "/var/faas/runtime/node-v12.2.0-linux-x64",
		Args: []string{
			"/var/runtime/bootstrap",
		},
	}
}
