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

package stubs

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/service/lambda"
	routing "github.com/qiangxue/fasthttp-routing"

	"github.com/baidu/openless/cmd/stubs/options"
	"github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/brn"
	"github.com/baidu/openless/pkg/util/json"
	"github.com/baidu/openless/pkg/util/logs"
)

var functionUid = "df391b08c64c426a81645468c75163a5"
var defaultZipFileNodejs01 = `UEsDBBQACAAIAAyjX00AAAAAAAAAAAAAAAAIABAAaW5kZXguanNVWAwAsJ/ZW/ie2Vv6Z7qeS60o
yC8qKdbLSMxLyUktUrBV0EgtS80r0VFIzs8rSa0AMRJzcpISk7M1FWztFKq5FIAAJqSRV5qTo6Og
5JGak5OvUJ5flJOiqKRpzVVrDQBQSwcILzRMjVAAAABYAAAAUEsDBAoAAAAAAHCjX00AAAAAAAAA
AAAAAAAJABAAX19NQUNPU1gvVVgMALSf2Vu0n9lb+me6nlBLAwQUAAgACAAMo19NAAAAAAAAAAAA
AAAAEwAQAF9fTUFDT1NYLy5faW5kZXguanNVWAwAsJ/ZW/ie2Vv6Z7qeY2AVY2dgYmDwTUxW8A9W
iFCAApAYAycQGwFxHRCD+BsYiAKOISFBUCZIxwIgFkBTwogQl0rOz9VLLCjISdXLSSwuKS1OTUlJ
LElVDggGKXw772Y0iO5J8tAH0QBQSwcIDgnJLFwAAACwAAAAUEsBAhUDFAAIAAgADKNfTS80TI1Q
AAAAWAAAAAgADAAAAAAAAAAAQKSBAAAAAGluZGV4LmpzVVgIALCf2Vv4ntlbUEsBAhUDCgAAAAAA
cKNfTQAAAAAAAAAAAAAAAAkADAAAAAAAAAAAQP1BlgAAAF9fTUFDT1NYL1VYCAC0n9lbtJ/ZW1BL
AQIVAxQACAAIAAyjX00OCcksXAAAALAAAAATAAwAAAAAAAAAAECkgc0AAABfX01BQ09TWC8uX2lu
ZGV4LmpzVVgIALCf2Vv4ntlbUEsFBgAAAAADAAMA0gAAAHoBAAAAAA==`

func installApiserver(router *routing.Router, options *options.StubsOptions) {
	routerGroup := router.Group("/v1")

	routerGroup.Get("/functions/<functionName>", GetFunctionHandler(options)).
		Post(CreateFunctionHandler(options))
	routerGroup.Get("/runtimes/<runtimeName>/configuration", GetRuntimeHandler(options))
}

func CreateFunctionHandler(options *options.StubsOptions) routing.Handler {
	return func(c *routing.Context) error {
		functionName := c.Param("functionName")
		logs.Infof("create function %s", functionName)

		function, codeData, err := createFunctionMeta(c)
		if err != nil {
			return ErrResponse(c, http.StatusInternalServerError, err)
		}

		funcDirPath, err := filepath.Abs(filepath.Join(options.FunctionDir, *function.Configuration.FunctionArn))
		if err != nil {
			return ErrResponse(c, http.StatusInternalServerError, err)
		}

		err = os.MkdirAll(funcDirPath, os.ModePerm)
		if err != nil {
			if !os.IsExist(err) {
				return ErrResponse(c, http.StatusInternalServerError, err)
			}
		}

		codePath := filepath.Join(funcDirPath, "code.zip")
		err = ioutil.WriteFile(codePath, codeData, os.ModePerm)
		if err != nil {
			return ErrResponse(c, http.StatusInternalServerError, err)
		}
		function.Code.SetLocation(codePath)

		funcConfigData, err := json.Marshal(function)
		if err != nil {
			return ErrResponse(c, http.StatusInternalServerError, fmt.Errorf("create function error: %v", err))
		}

		metaPath := filepath.Join(funcDirPath, "meta.json")
		err = ioutil.WriteFile(metaPath, funcConfigData, os.ModePerm)
		if err != nil {
			return ErrResponse(c, http.StatusInternalServerError, err)
		}

		c.SetStatusCode(http.StatusOK)
		return nil
	}
}

func GetFunctionHandler(options *options.StubsOptions) routing.Handler {
	return func(c *routing.Context) error {
		functionBrn := c.Param("functionName")
		logs.Infof("get function %s", functionBrn)
		accountID := string(c.Request.Header.Peek(api.HeaderXAccountID))
		if accountID == "" {
			 accountID = functionUid
		}
		hashedAccountID := brn.Md5BceUid(accountID)
		functionName, version, _, err := brn.DealFName(hashedAccountID, functionBrn)
		if err != nil {
			return ErrResponse(c, http.StatusBadRequest, err)
		}

		if version == "" {
			functionBrn = brn.GenerateFuncBrnString("bj", functionUid, functionName, "$LATEST")
		}

		metaPath, err := filepath.Abs(filepath.Join(options.FunctionDir, functionBrn, "meta.json"))
		if err != nil {
			return ErrResponse(c, http.StatusInternalServerError, err)
		}

		metaFile, err := os.Open(metaPath)
		if err != nil {
			if os.IsNotExist(err) {
				return ErrResponse(c, http.StatusNotFound, fmt.Errorf("not found function "+functionBrn))
			}
		}

		metaData, err := ioutil.ReadAll(metaFile)
		if err != nil {
			return ErrResponse(c, http.StatusInternalServerError, err)
		}

		c.SetStatusCode(http.StatusOK)
		c.Write(metaData)
		return nil
	}
}

func GetRuntimeHandler(options *options.StubsOptions) routing.Handler {
	return func(c *routing.Context) error {
		logs.Infof("get runtime")
		runtimeName := c.Param("runtimeName")
		runtimeInfo, ok := runtimesMap[runtimeName]
		if !ok {
			return ErrResponse(c, http.StatusInternalServerError, fmt.Errorf("invaild runtime %s", runtimeName))
		}
		confData, err := json.Marshal(runtimeInfo)
		if err != nil {
			return ErrResponse(c, http.StatusInternalServerError, err)
		}

		c.SetStatusCode(http.StatusOK)
		c.Write(confData)
		return nil
	}
}

func ErrResponse(c *routing.Context, code int, err error) error {
	logs.Errorf("response code %d, msg: %v", code, err)
	c.SetStatusCode(code)
	c.WriteString(err.Error())

	return nil
}

func createFunctionMeta(c *routing.Context) (*api.GetFunctionOutput, []byte, error) {
	args := CreateFunctionArgs{}
	body := c.PostBody()
	if len(body) > 0 {
		if err := json.Unmarshal(body, &args); err != nil {
			return nil, nil, err
		}
	}

	function := &api.GetFunctionOutput{
		Code:          &api.FunctionCodeLocation{},
		Concurrency:   &api.Concurrency{},
		Configuration: &api.FunctionConfiguration{},
		LogConfig:     &api.LogConfiguration{},
		Tags:          map[string]*string{},
	}

	function.Code.SetRepositoryType("filesystem")
	function.Configuration.Uid = functionUid
	function.Configuration.PodConcurrentQuota = 0
	function.Configuration.SetEnvironment(&lambda.EnvironmentResponse{})
	function.Configuration.SetTimeout(5)
	function.Configuration.SetMemorySize(128)
	function.Configuration.SetRuntime("nodejs8.5")
	function.Configuration.SetHandler("index.handler")
	function.Configuration.SetVersion("$LATEST")
	function.Configuration.SetCommitID("349dd8a4-db7d-4fc3-a4cd-8c8b482e00c3")
	function.Configuration.SetLastModified(time.Now().Format(time.RFC3339))

	if args.PodConcurrentQuota > 0 {
		function.Configuration.PodConcurrentQuota = args.PodConcurrentQuota
	}

	if args.Timeout > 0 {
		function.Configuration.SetTimeout(args.Timeout)
	}

	if args.MemorySize > 0 {
		function.Configuration.SetMemorySize(args.MemorySize)
	}

	if len(args.Runtime) > 0 {
		function.Configuration.SetRuntime(args.Runtime)
	}

	if len(args.Handler) > 0 {
		function.Configuration.SetHandler(args.Handler)
	}

	if len(args.Version) > 0 {
		function.Configuration.SetVersion(args.Version)
	}

	if len(args.Description) > 0 {
		function.Configuration.SetDescription(args.Description)
	}

	fName := c.Param("functionName")
	accountID := string(c.Request.Header.Peek(api.HeaderXAccountID))
	if accountID == "" {
		accountID = functionUid
	}
	hashedAccountID := brn.Md5BceUid(accountID)
	functionName, version, _, err := brn.DealFName(hashedAccountID, fName)
	if err != nil {
		return nil, nil, err
	}

	if version != "" {
		function.Configuration.SetVersion(version)
	}

	function.Configuration.SetFunctionName(functionName)
	function.Configuration.SetFunctionArn(brn.GenerateFuncBrnString("bj", functionUid, functionName, *function.Configuration.Version))

	codeStr := args.Code
	if len(codeStr) == 0 {
		codeStr = defaultZipFileNodejs01
	}

	codeData, err := base64.StdEncoding.DecodeString(codeStr)
	if err != nil {
		return nil, nil, err
	}

	hasher := sha256.New()
	hasher.Write(codeData)
	codeSha := base64.StdEncoding.EncodeToString(hasher.Sum(nil))
	function.Configuration.SetCodeSha256(codeSha)
	function.Configuration.SetCodeSize(int64(len(codeData)))

	return function, codeData, nil
}
