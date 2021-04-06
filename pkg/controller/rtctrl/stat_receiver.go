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
	"io"
	"sync"

	_ "github.com/baidu/openless/pkg/userlog/json"
	_ "github.com/baidu/openless/pkg/userlog/plain"
	"github.com/baidu/openless/pkg/util/json"
	"github.com/baidu/openless/pkg/util/logs"
)

type statInfo struct {
	PodName string `json:"podname,omitempty"`
	MemUsed int64  `json:"memory,omitempty"`
}

type statinfoReceiver struct {
	name string
	conn io.ReadWriter
}

func (r *statinfoReceiver) Recv(callbk func(*statInfo) error) {
	reader := bufio.NewReader(r.conn)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			logs.V(5).Infof("recv %s status failed: %s", r.name, err.Error())
			break
		}

		stat := &statInfo{}
		err = json.Unmarshal(line, stat)
		if err != nil {
			logs.V(4).Warnf("recv %s status failed: %s", r.name, err.Error())
			break
		}
		if len(stat.PodName) == 0 {
			stat.PodName = r.name
		}
		if err = callbk(stat); err != nil {
			logs.V(4).Warnf("write %s failed: %s", r.name, err.Error())
		}
	}
}

type statsConn struct {
	name    string
	conn    io.ReadWriteCloser
	memused int64
}

type statsMap struct {
	sync.Map
}

func (m *statsMap) get(runtimeid string) *statsConn {
	v, ok := m.Load(runtimeid)
	if ok {
		return v.(*statsConn)
	}
	return nil
}

func (m *statsMap) set(runtimeid string, sc *statsConn) {
	m.Store(runtimeid, sc)
}

func (m *statsMap) del(runtimeid string, sc *statsConn) {
	v, ok := m.Load(runtimeid)
	if ok {
		old := v.(*statsConn)
		if old == sc {
			m.Delete(runtimeid)
		}
	}
}
