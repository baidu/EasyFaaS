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
	"testing"
	"time"

	"github.com/baidu/openless/pkg/util/id"

	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/util/logs"
)

func TestStatisticsInfo(t *testing.T) {
	cmid := "commit-xxx"
	funcName := "test"
	reqid := id.GetRequestID()
	now := time.Now().UnixNano()
	ver := "1"
	timeout := int64(3)
	reqinfo := RequestInfo{
		RequestID:         reqid,
		InvokeStartTimeNS: now,
		InvokeStartTimeMS: now / int64(time.Millisecond),
		SyncChannel:       make(chan struct{}, 1),
		TimeoutChannel:    make(chan struct{}, 1),
		Status:            StatusSuccess,
		Input: &InvocationInput{
			RequestID: reqid,
			Configuration: &api.FunctionConfiguration{
				CommitID: &cmid,
				FunctionConfiguration: lambda.FunctionConfiguration{
					Version:      &ver,
					Timeout:      &timeout,
					FunctionName: &funcName,
				},
			},
			User: &api.User{
				ID: "xxx",
			},
			WithStreamMode: true,
			Request: &api.InvokeProxyRequest{
				Headers: make(map[string]string, 0),
				Body:    []byte("{}"),
			},
			Response:      api.NewInvokeProxyResponse(),
			EnableMetrics: false,
			Logger:        logs.NewLogger().WithField("request_id", reqid),
		},
		Output: &InvocationOutput{Output: &InvocationResponse{}, Statistic: &InvocationStatistic{}},
	}
	info := statisticsInfo(&reqinfo)
	if info == nil {
		t.Error("statistics info should not be nil")
		return
	}
	if err := info.Decode(info.Encode()); err != nil {
		t.Errorf("encode/decode statistic info failed: %s", err)
	}
}
