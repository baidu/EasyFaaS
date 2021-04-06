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

package logs_test

import (
	"testing"
	"time"

	"github.com/baidu/openless/pkg/util/logs"
)

func TestTimeTrackWithLogger(t *testing.T) {
	logger := logs.NewLogger()

	defer logger.TimeTrack(time.Now(), "testTook")
	time.Sleep(1 * time.Second)
}

func TestTimeTrack(t *testing.T) {
	type args struct {
		start time.Time
		name  string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "timetrack",
			args: args{
				start: time.Now(),
				name:  "hello",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs.TimeTrack(tt.args.start, tt.args.name)
		})
	}
}
