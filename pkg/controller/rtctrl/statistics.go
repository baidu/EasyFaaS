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
	"encoding/base64"
	"net/http"
	"time"

	"github.com/baidu/openless/pkg/util/json"
)

type StatisticInfo struct {
	UserID     string  `json:"userid"`
	RequestID  string  `json:"reqid,omitempty"`
	Function   string  `json:"function"`
	Version    string  `json:"version"`
	StartTime  int64   `json:"startms"`
	Duration   float64 `json:"duration"`
	MemoryUsed int64   `json:"memused"`
	StatusCode int     `json:"status"`
}

func (si *StatisticInfo) Encode() string {
	data, _ := json.Marshal(si)
	return base64.StdEncoding.EncodeToString(data)
}

func (si *StatisticInfo) Decode(str string) error {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, si)
	if err != nil {
		return err
	}
	return nil
}

func statisticsInfo(request *RequestInfo) *StatisticInfo {
	if nil == request {
		return nil
	}
	input := request.Input
	funcname := ""
	if nil != input.Configuration.FunctionName {
		funcname = *(input.Configuration.FunctionName)
	}
	version := ""
	if nil != input.Configuration.Version {
		version = *(input.Configuration.Version)
	}
	now := time.Now().UnixNano()
	statusCode := http.StatusExpectationFailed
	if request.Status == StatusSuccess {
		statusCode = http.StatusOK
	} else if request.Status == StatusTimeout {
		statusCode = http.StatusRequestTimeout
	}
	msg := StatisticInfo{
		UserID:     request.Input.User.ID,
		Function:   funcname,
		Version:    version,
		StartTime:  request.InvokeStartTimeMS,
		Duration:   float64(now-request.InvokeStartTimeNS) / float64(time.Millisecond),
		MemoryUsed: request.MaxMemUsedBytes,
		StatusCode: statusCode,
	}

	return &msg
}
