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
	"fmt"
	"io"

	"github.com/baidu/openless/pkg/util/json"
	"github.com/baidu/openless/pkg/util/logs"
)

// startRunnerLoop
func (info *RuntimeInfo) startRunnerLoop(params *startRunnerParams) error {
	if err := info.initRunner(params); err != nil {
		return err
	}
	info.recvRunnerInfoLoop(params)
	return nil
}

// recvRunnerInfoLoop
func (info *RuntimeInfo) recvRunnerInfoLoop(params *startRunnerParams) {
	defer func() {
		logs.Infof("try to stop runner %s", info.RuntimeID)
		info.stopRunner()
		logs.Infof("runner %s stopped", info.RuntimeID)
	}()
	decoder := json.NewDecoder(info.runnerConn)
	if params.buf != nil {
		decoder = json.NewDecoder(params.buf)
	}
	var ci ReportedRunnerInfo
	for {
		info.runnerUpdate()
		err := decoder.Decode(&ci)
		if err != nil {
			if err == io.EOF {
				logs.Infof("runner %s read EOF", info.RuntimeID)
			} else {
				logs.Errorf("runner %s read failed: %s", info.RuntimeID, err.Error())
			}
			break
		}
		// TODO: update runner status
	}
}

// initRunner
func (info *RuntimeInfo) initRunner(params *startRunnerParams) error {
	info.RebootWait()
	logs.V(5).Infof("runner %s is rebooting", info.RuntimeID)

	info.invokeLock.Lock()
	defer info.invokeLock.Unlock()

	if info.State != RuntimeStateClosed {
		logs.Errorf("runtime %s  current states is %s", info.RuntimeID, info.State)
		return fmt.Errorf("duplicate runner")
	}

	info.SetState(RuntimeStateCold)
	info.runnerConn = params.conn
	info.Abnormal = false
	return nil
}

// stopRunner
func (info *RuntimeInfo) stopRunner() {
	info.runtimeWaitGroup.Wait()

	info.invokeLock.Lock()
	defer info.invokeLock.Unlock()

	info.runnerConn.Close()
	// suspect or check: https://github.com/golang/go/issues/28446
	if info.State == RuntimeStateMerged || info.State == RuntimeStateReclaiming {
		return
	}
	info.SetState(RuntimeStateClosed)
}

// runnerUpdate
func (info *RuntimeInfo) runnerUpdate() {
	info.invokeLock.Lock()
	defer info.invokeLock.Unlock()

	info.updateLastLivenessTime()
}
