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
	"bufio"
	"fmt"
	"sync"

	"github.com/baidu/openless/pkg/util/logs"
)

const (
	ZeroByte byte = '\000'
	TabByte  byte = '\t'
	LineByte byte = '\n'

	OpenlessSysLog = 0
	StdoutLog      = 1
	StderrLog      = 2
)

var (
	defaultLogBufferLength = 64 * 1024 // default 64KB log buffer size
)

type stdLogDispatcher struct {
	runtimeID    string
	logStoreMap  *logStatStoreMap
	logFrom      int
	name         string
	conn         *bufio.ReadWriter
	bufferLength int
	logger       *logs.Logger
}

func (lsm *logStatStoreMap) NewStdLogDispatcher(runtimeID string, logFrom int, conn *bufio.ReadWriter, bufferLength int) *stdLogDispatcher {
	name := fmt.Sprintf("%s stdlog.%d", runtimeID, logFrom)
	if bufferLength <= 0 {
		bufferLength = defaultLogBufferLength
	}
	return &stdLogDispatcher{
		runtimeID:    runtimeID,
		logStoreMap:  lsm,
		logFrom:      logFrom,
		name:         name,
		conn:         conn,
		bufferLength: bufferLength,
		logger:       logs.NewLogger().WithField("name", name),
	}
}

func (d *stdLogDispatcher) Read() {
	buf := make([]byte, d.bufferLength+1)
	next := 0     // offset
	lnTabCnt := 0 // tab count in recent line
	var lnReqID string
	var ls LogStatStore
	for {

		// bytes count from connection
		read, err := d.conn.Read(buf[next:d.bufferLength])
		d.logger.V(9).Infof("recv res: read %d, data %s, err %+v", read, string(buf[next:next+read]), err)
		if err != nil {
			d.logger.V(5).Errorf("recv failed: %s", err.Error())
			return
		}

		// if log store map is empty, discard the data
		if len(d.logStoreMap.storeMap) == 0 {
			d.logger.Warn("can't find log store map")
			next = 0
			continue
		}

		// if can't get the log store of the last request, discard the data
		ls, err = d.logStoreMap.getLast()
		if err != nil {
			d.logger.Warnf("get last log store of %s failed", d.logStoreMap.String())
			next = 0
			continue
		}
		if ls == nil {
			next = 0
			continue
		}

		next += read

		nzero := 0
		lastln := -1
		lastTab := -1

		// parse data
		for i := 0; i < next; i++ {
			if buf[i] == ZeroByte { // \0 means the end of the request
				d.logger.V(9).Infof("next %d lastln %d i %d nzero %d", next, lastln, i, nzero)
				nzero++
				copy(buf[i:], buf[i+1:next]) // remove \0
				i--
				next--
				nwrite, err := ls.WriteStdLog(d.logFrom, buf[lastln+1:i+1], true)
				// write error: discard the log data, start the new line process
				// write all success: start the new line process
				if err != nil || nwrite == i-lastln {
					if err != nil {
						d.logger.Errorf("write std log %s len %d with eof lastln %d i %d failed: %s", ls.String(), nwrite, lastln, i, err)
					} else {
						d.logger.V(6).Infof("write std log %s len %d with eof lastln %d i %d", ls.String(), nwrite, lastln, i)
					}
					lastln = i
					lnTabCnt = 0
					lastTab = -1
					continue
				}
				// write partial success: adjust the next pointer
				d.logger.Warnf("write partial std log %s len %d with eof lastln %d i %d", ls.String(), nwrite, lastln, i)
				i = lastln + nwrite
			} else if buf[i] == LineByte {
				d.logger.V(9).Infof("next %d lastln %d i %d nzero %d", next, lastln, i, nzero)
				nwrite, err := ls.WriteStdLog(d.logFrom, buf[lastln+1:i+1], false)
				// write error: discard the log data, start the new line process
				// write all success: start the new line process
				if err != nil || nwrite == i-lastln {
					if err != nil {
						d.logger.Errorf("write std log %s len %d lastln %d i %d failed: %s", ls.String(), nwrite, lastln, i, err)
					} else {
						d.logger.V(6).Infof("write std log %s len %d lastln %d i %d", ls.String(), nwrite, lastln, i)
					}
					// mark as new line process
					lastln = i
					lnTabCnt = 0
					lastTab = -1
					continue
				}
				// write partial success: adjust the next pointer
				d.logger.Warnf("write partial std log %s len %d lastln %d i %d", ls.String(), nwrite, lastln, i)
				i = lastln + nwrite
			} else if buf[i] == TabByte {
				d.logger.V(9).Infof("next %d lastln %d i %d nzero %d", next, lastln, i, nzero)
				lnTabCnt++
				if lnTabCnt == 2 {
					lnReqID = string(buf[lastTab+1 : i])
					d.logger.V(6).Infof("parse request id %s", lnReqID)
					lls := d.logStoreMap.get(lnReqID)
					if lls != nil {
						ls = lls
					} else {
						d.logger.Errorf("map %s can't find log store of request %s", ls.String(), lnReqID)
					}
				}
				lastTab = i
			}
		}

		// Parse long data:
		// when there are no \0 character and no new line, check the buffer size
		// when buffer is full, add \n character; otherwise, continue read from buffer
		if nzero == 0 && lastln < 0 {
			if next == d.bufferLength {
				buf[next] = LineByte
				nwrite, err := ls.WriteStdLog(d.logFrom, buf[lastln+1:next+1], false)
				if err != nil || nwrite == next-lastln {
					if err != nil {
						d.logger.Errorf("buffer full write std log len %d err %s with eof lastln %d next %d", nwrite, err, lastln, next, err)
					} else {
						d.logger.V(6).Infof("buffer full write std log len %d err %s with eof lastln %d next %d", nwrite, err, lastln, next)
					}
					next = 0
				} else if nwrite > 0 {
					d.logger.Warnf("buffer full write partial std log len %d err %s with eof lastln %d next %d", ls.String(), nwrite, lastln, next)
					next -= nwrite
					copy(buf[0:], buf[nwrite:next+1])
				}
				lastln = -1
				lastTab = -1
				lnTabCnt = 0
			} else {
				continue
			}
		}

		d.logger.V(9).Infof("next %d lastln %d nzero %d", next, lastln, nzero)
		copy(buf[0:], buf[lastln+1:next])
		next = next - lastln - 1
		lastln = -1
		lastTab = -1
		lnTabCnt = 0
	}
}

type logStatStoreMap struct {
	runtimeID string
	lastReqID string
	storeMap  map[string]LogStatStore
	lock      sync.RWMutex
}

func newLogStatStoreMap(runtimeID string) *logStatStoreMap {
	return &logStatStoreMap{
		runtimeID: runtimeID,
		storeMap:  make(map[string]LogStatStore, 0),
		lock:      sync.RWMutex{},
	}
}

func (ls *logStatStoreMap) set(requestID string, store LogStatStore) {
	ls.lock.Lock()
	defer ls.lock.Unlock()
	ls.storeMap[requestID] = store
	ls.lastReqID = requestID
}

func (ls *logStatStoreMap) get(requestID string) (store LogStatStore) {
	ls.lock.RLock()
	defer ls.lock.RUnlock()
	store, ok := ls.storeMap[requestID]
	if !ok {
		return nil
	}
	return store
}

func (ls *logStatStoreMap) getLast() (store LogStatStore, err error) {
	ls.lock.RLock()
	defer ls.lock.RUnlock()
	if ls.lastReqID != "" {
		store, ok := ls.storeMap[ls.lastReqID]
		if !ok {
			err = fmt.Errorf("get log store of last request id %s failed", ls.lastReqID)
			logs.Errorf("log stat store get last failed: %s", err)
			return nil, err
		}
		return store, nil
	}
	return nil, nil
}

func (ls *logStatStoreMap) del(requestID string, store LogStatStore) {
	ls.lock.Lock()
	defer ls.lock.Unlock()
	old, ok := ls.storeMap[requestID]
	if !ok {
		return
	}
	if old == store {
		delete(ls.storeMap, requestID)
		ls.lastReqID = ""
	}
	return
}

func (ls *logStatStoreMap) String() string {
	ls.lock.RLock()
	defer ls.lock.RUnlock()
	return fmt.Sprintf("logStoreMap-%s", ls.runtimeID)
}

type storeMap struct {
	sync.Map
}

func (m *storeMap) setNXMap(runtimeID string) {
	v, ok := m.Load(runtimeID)
	var needInit bool
	if ok {
		_, ok := v.(*logStatStoreMap)
		if !ok {
			needInit = true
		}
	} else {
		needInit = true
	}
	if needInit {
		m.Store(runtimeID, newLogStatStoreMap(runtimeID))
	}
	return
}

func (m *storeMap) getMap(runtimeID string) (storeMap *logStatStoreMap, ok bool) {
	v, ok := m.Load(runtimeID)
	if ok {
		storeMap, ok = v.(*logStatStoreMap)
		return
	}
	return
}

func (m *storeMap) delMap(runtimeID string) {
	m.Delete(runtimeID)
}

func (m *storeMap) get(runtimeID, requestID string) LogStatStore {
	v, ok := m.Load(runtimeID)
	if !ok {
		return nil
	}
	sm, ok := v.(*logStatStoreMap)
	if !ok {
		return nil
	}
	return sm.get(requestID)
}

func (m *storeMap) set(runtimeID, requestID string, store LogStatStore) (err error) {
	v, ok := m.Load(runtimeID)
	var newMap bool
	var sm *logStatStoreMap
	if ok {
		sm, ok = v.(*logStatStoreMap)
		if !ok {
			newMap = true
			sm = newLogStatStoreMap(runtimeID)
		}
	} else {
		newMap = true
		sm = newLogStatStoreMap(runtimeID)
	}
	sm.set(requestID, store)
	if newMap {
		m.Store(runtimeID, sm)
	}
	return
}

func (m *storeMap) del(runtimeID, requestID string, store LogStatStore) {
	v, ok := m.Load(runtimeID)
	if ok {
		sm, ok := v.(*logStatStoreMap)
		if !ok {
			return
		}
		sm.del(requestID, store)
		return
	}
	return
}
