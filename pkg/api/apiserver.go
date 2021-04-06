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

package api

import (
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/lambda"
)

var (
	RegfunctionName = regexp.MustCompile("^[a-zA-Z0-9-_]+$")
	RegVersion      = regexp.MustCompile("^(\\$LATEST|([0-9]+))$")
)

// CodeStorage
type CodeStorage struct {
	Location       string
	RepositoryType string
}

// FunctionConfig
type FunctionConfig struct {
	CodeSha256   string
	CodeSize     int32
	FunctionArn  string
	FunctionName string
	Handler      string
	Version      string
	Runtime      string
	MemorySize   *int
	Timeout      *int
	Environment  *Environment
	CommitID     *string `json:"CommitId"`

	LogType   string `json:",omitempty"`
	LogBosDir string `json:",omitempty"`

	PodConcurrentQuota *int `json:"PodConcurrentQuota"`
}

// Function Environment
type Environment struct {
	Variables map[string]string
}

// RuntimeConfiguration
type RuntimeConfiguration struct {
	Name string
	Bin  string
	Path string
	Args []string
}

// GetFunctionInput
type GetRuntimeConfigurationInput struct {
	Authorization string
	RuntimeName   string
	RequestID     string
}

// GetFunctionInput
type GetFunctionInput struct {
	Authorization string
	lambda.GetFunctionInput
	RequestID  string
	AccountID  string
	WithCache  bool
	SimpleAuth bool
}

// FunctionCodeLocation
type FunctionCodeLocation struct {
	lambda.FunctionCodeLocation
	LogType string
}

// Concurrency xxx
type Concurrency struct {
	lambda.PutFunctionConcurrencyOutput
	AccountReservedSum int
}

type LogConfiguration struct {
	LogType string

	// for bos
	BosDir string

	// for other
	Params string
}

func (c LogConfiguration) String() string {
	return awsutil.Prettify(c)
}

// FunctionConfiguration
type FunctionConfiguration struct {
	lambda.FunctionConfiguration
	CommitID           *string `json:"CommitId,omitempty"`
	Uid                string  `json:",omitempty"`
	LogType            string  `json:",omitempty"`
	LogBosDir          string  `json:",omitempty"`
	PodConcurrentQuota uint64  `json:",omitempty"`
}

func IsNoneLogType(logType string) bool {
	if logType == "" || logType == "none" {
		return true
	}
	return false
}

func (s *FunctionConfiguration) String() string {
	return awsutil.Prettify(s)
}

func (s *FunctionConfiguration) SetCommitID(v string) *FunctionConfiguration {
	s.CommitID = &v
	return s
}

// GetFunctionOutput
type GetFunctionOutput struct {
	//_ struct{} `type:"structure"`

	// The object for the Lambda function location.
	Code *FunctionCodeLocation `type:"structure"`

	// The concurrent execution limit set for this function. For more information,
	// see concurrent-executions.
	Concurrency *Concurrency `type:"structure",json:",omitempty"`

	// A complex type that describes function metadata.
	Configuration *FunctionConfiguration `type:"structure"`
	LogConfig     *LogConfiguration      `type:"structure",json:",omitempty"`

	Uid                string  `json:",omitempty"`
	LogType            string  `json:",omitempty"`
	PodConcurrentQuota *uint64 `json:",omitempty"`
	// Returns the list of tags associated with the function.
	Tags map[string]*string `type:"map",json:",omitempty"`
}

// GetFunctionInput
type GetAliasInput struct {
	FunctionBrn   string
	FunctionName  string
	Qualifier     string
	Authorization string
	RequestID     string
	AccountID     string
	WithCache     bool
	SimpleAuth    bool
}

type GetAliasOutput = Alias

type Alias struct {
	Id                      uint `json:"-"`
	AliasBrn                string
	AliasArn                string
	FunctionName            string
	FunctionVersion         string
	Name                    string
	Description             *string
	Uid                     string
	UpdatedAt               time.Time
	CreatedAt               time.Time
	AdditionalVersion       *string
	AdditionalVersionWeight *float64
}
