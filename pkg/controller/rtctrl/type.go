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

package rtctrl

import (
	"bufio"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

type RuntimeStateType = string

const (
	RuntimeStateCold       RuntimeStateType = "cold"
	RuntimeStateWarmUp     RuntimeStateType = "warmup"
	RuntimeStateWarm       RuntimeStateType = "warm"
	RuntimeStateMerged     RuntimeStateType = "merged"
	RuntimeStateStopping   RuntimeStateType = "stopping"
	RuntimeStateStopped    RuntimeStateType = "stopped"
	RuntimeStateClosed     RuntimeStateType = "closed"
	RuntimeStateReclaiming RuntimeStateType = "reclaiming"
)

type NewRuntimeParameters struct {
	RuntimeID               string
	ConcurrentMode          bool
	StreamMode              bool
	WaitRuntimeAliveTimeout int
	IsFrozen                bool
	Resource                *api.Resource
}

type ReportedRunnerInfo struct {
	Hostname    string
	ContainerID string
	Status      string
}

type startRuntimeParams struct {
	commitID   string
	hostIP     string
	conn       net.Conn
	warmNotify chan struct{}
	urlParams  url.Values
}

type startRunnerParams struct {
	conn net.Conn
	buf  *bufio.ReadWriter
}

type InvokeRequest struct {
	RequestID       string `json:"requestid"`
	Version         string `json:"version"`
	AccessKeyID     string `json:"accessKey"`
	AccessKeySecret string `json:"secretKey"`
	SecurityToken   string `json:"securityToken"`
	ClientContext   string `json:"clientContext,omitempty"`
	EventObject     string `json:"eventObject,omitempty"`
}

type InvokeHTTPRequest struct {
	RequestID       string
	Request         *http.Request
	FunctionTimeout int64
	Logger          *logs.Logger
	CtxCancel       func()
}

type InvokeHTTPResponse struct {
	RequestID string
	Response  *http.Response
}

type InvokeResponse struct {
	RequestID  string `json:"requestid"`
	Success    bool   `json:"success"`
	FuncResult string `json:"result,omitempty"`
	FuncError  string `json:"error,omitempty"`
}

type RuntimeInfo struct {
	RuntimeID  string `json:"RuntimeID"`
	invokeLock sync.Mutex
	rebootLock sync.Mutex

	// runtime state machine
	State         RuntimeStateType `json:"State"`
	Used          bool             `json:"used"`
	Marked        bool             `json:"marked"`
	Abnormal      bool             `json:"abnormal"`
	AbnormalTimes uint             `json:"abnormalTimes"`

	// runtime resource
	Resource *api.Resource `json:"Resource"`

	// Function meta
	UserID                  string `json:"UserID"` // CFC User ID
	CommitID                string `json:"CommitID"`
	MemorySize              uint64 `json:"MemorySize"`
	ConcurrentMode          bool   `json:"ConcurrentMode"`
	DefaultConcurrentMode   bool   `json:"DefaultConcurrentMode"`
	Concurrency             uint64 `json:"Concurrency"`
	WithStreamMode          bool   `json:"WithStreamMode"` // is http stream mode
	WaitRuntimeAliveTimeout int    `json:"WaitRuntimeAliveTimeout"`

	// Statistics
	PreLoadTimeMS  int64 `json:"PreLoadTimeMS"`
	PostLoadTimeMS int64 `json:"PostLoadTimeMS"`
	PreInitTimeMS  int64 `json:"PreInitTimeMS"`
	PostInitTimeMS int64 `json:"PostInitTimeMS"`
	AcceptReqCnt   int64 `json:"AcceptReqCnt"`
	RejectReqCnt   int64 `json:"RejectReqCnt"`

	LastLivenessTime time.Time `json:"LastLivenessTime"`
	LastAccessTime   time.Time `json:"LastAccessTime"`
	LastResetTime    time.Time `json:"LastResetTime"`

	// runnerConn: runner connection
	// runner connection is used to health check
	runnerConn net.Conn
	// runtimeConn: runtime connection
	// In general mode, runtime connection is used to send request and receive response
	runtimeConn net.Conn
	// requestChan: request queue
	// In general mode, the request of invocation will be transfer into InvokeRequest
	// The runtime background goroutine will consume the InvokeRequest
	requestChan chan *InvokeRequest

	// httpClient: http stream mode client
	httpClient *http.Client
	// httpRequestChan: http request queue
	// In http stream mode, the request of invocation will be transfer into http request
	// The runtime background goroutine will consume the request
	httpRequestChan chan *InvokeHTTPRequest
	// httpResponseChan
	httpResponseChan chan *InvokeHTTPResponse
	retryDeadline    time.Duration

	requestMap sync.Map
	// rebootWaitGroup: runner reboot wait group
	// wait for cooldown/reborn process to finish
	rebootWaitGroup sync.WaitGroup
	// runtimeRunChan
	// notify the request waiting list that runtime can handler the requests
	runtimeRunChan chan struct{}
	// runtimeStopChan
	// notify the background goroutine that runtime is going to stop
	runtimeStopChan chan struct{}

	runtimeStoppingChan chan struct{}

	// runtimeWaitGroup
	// waiting for all the background goroutines to finish
	runtimeWaitGroup sync.WaitGroup
}

// InvocationInput function call input param
type InvocationInput struct {
	// runtime
	Runtime *RuntimeInfo

	// ExternalRequestID: external request ID
	ExternalRequestID string

	// RequestID: internal request ID
	RequestID string

	// User xxx
	User *api.User

	// The object for the Lambda function location.
	Code *api.FunctionCodeLocation

	// A complex type that describes function metadata.
	Configuration *api.FunctionConfiguration

	// log configuration
	LogConfig *api.LogConfiguration

	// define whether transfer request body as a stream
	WithStreamMode bool

	Request  *api.InvokeProxyRequest
	Response *api.InvokeProxyResponse

	// Enable Metric
	EnableMetrics bool

	// IsLogTail
	IsLogTail bool

	Logger      *logs.Logger
	InvokeType  string
	TriggerType string
}

type InvocationOutput struct {
	Output    *InvocationResponse
	Statistic *InvocationStatistic
}

// InvocationOutput function call output param
type InvocationResponse struct {
	FuncResult string   `json:"result,omitempty"`
	LogMessage []string `json:"log"`
	FuncError  string   `json:"errtype,omitempty"`
	ErrorInfo  string   `json:"errinfo,omitempty"`
	Response   *api.InvokeProxyResponse
}

type InvocationStatistic struct {
	Statistic *StatisticInfo
	Metric    *RtCtrlInvokeMetric
}

type UserLogType string

const (
	UserLogTypePlain UserLogType = "plain"
	UserLogTypeJson  UserLogType = "json"
)

func (u UserLogType) Valid() bool {
	if u == UserLogTypePlain || u == UserLogTypeJson {
		return true
	}
	return false
}

const (
	userLogSingleFile = "controller-userlog.log"
)
