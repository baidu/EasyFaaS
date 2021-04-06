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
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/baidu/openless/pkg/userlog"
	"github.com/baidu/openless/pkg/util/logs"
)

type LogStatStore interface {
	Receiver() string
	String() string
	WriteStdLog(from int, buf []byte, eof bool) (int, error)
	WriteFunctionLog(log string) error
	WriteFunctionReportLog(log string, params *reportParameters) error
	SetMemUsed(used int64)
	LogFile() string
	MemUsed() int64
	LogDone(set bool) bool
	Close() (string, error)
	Wait()
}

type bits uint8

const (
	flagClosed bits = 1 << iota
	flagOutdone
	flagErrdone
)

func (b *bits) Set(f bits) {
	*b = *b | f
}

func (b *bits) Has(f bits) bool {
	return *b&f != 0
}

type kunLogStatStore struct {
	requestID   string
	triggerType string
	runtimeID   string
	userID      string
	funcName    string
	funcBrn     string
	funcVersion string
	maxMem      int64

	mutex   sync.Mutex
	waitg   sync.WaitGroup
	flags   bits // outdone, errdone, closed
	remain  int
	logbuf  *bytes.Buffer
	fpath   string
	logfile userlog.UserLogFile
}

const (
	defaultUserLogLength = 4 * 1024
	maxUserLogSize       = 6 * 1024 * 1024
)

var (
	logBufPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, defaultUserLogLength))
		},
	}

	logSource         = []string{"faas", "stdout", "stderr"}
	errReceiverClosed = errors.New("buffer has closed")
)

func getUserLogPath(requestID, runtimeID, userID, funcName, funcVer, fpath, logtype string) string {
	var logpath string

	if logtype != string(UserLogTypePlain) {
		if _, err := os.Stat(fpath); err != nil {
			return ""
		}
		logpath = path.Join(fpath, userLogSingleFile)
		return logpath
	}

	logpath = path.Join(fpath, fmt.Sprintf("%s.%s.%s", userID, funcName, strings.TrimLeft(funcVer, "$")))
	_, err := os.Stat(logpath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(logpath, 0755)
			if err != nil {
				logs.V(4).Warn(fmt.Sprintf("recvlog %s create dir %s failed: %s",
					runtimeID, logpath, err.Error()), zap.String("request_id", requestID))
				logpath = ""
			}
		} else {
			logs.V(4).Warn(fmt.Sprintf("recvlog %s stat dir %s failed: %s",
				runtimeID, logpath, err.Error()), zap.String("request_id", requestID))
			logpath = ""
		}
	}
	if len(logpath) > 0 {
		logpath = path.Join(logpath, fmt.Sprintf("%d.%s", time.Now().UnixNano(), requestID))
		_, err = os.Stat(logpath)
		if err == nil {
			logs.V(4).Warn(fmt.Sprintf("recvlog %s logfile %s already exist",
				runtimeID, logpath), zap.String("request_id", requestID))
			logpath = ""
		}
	}
	return logpath
}

type LogStatStoreParameter struct {
	RequestID       string
	TriggerType     string
	RuntimeID       string
	UserID          string
	FunctionName    string
	FunctionBrn     string
	FunctionVersion string
	FilePath        string
	LogType         string
}

//func newLogStatStore(requestID, runtimeID, userID, funcName, funcVer, fpath, logtype string) LogStatStore {
func newLogStatStore(params *LogStatStoreParameter) LogStatStore {
	logpath := getUserLogPath(params.RequestID, params.RuntimeID, params.UserID, params.FunctionName, params.FunctionVersion, params.FilePath, params.LogType)
	logfile, err := userlog.CreateLogWriter(params.LogType, logpath, maxUserLogSize)
	if err != nil {
		logpath = ""
		if params.LogType == "" || params.LogType == "none" {
			logs.Info(fmt.Sprintf("create logfile failed: %v", err),
				zap.String("request_id", params.RequestID))
		} else {
			logs.V(4).Warn(fmt.Sprintf("create logfile failed: %v", err),
				zap.String("request_id", params.RequestID))
		}
	}
	buf := logBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	r := &kunLogStatStore{
		requestID:   params.RequestID,
		runtimeID:   params.RuntimeID,
		triggerType: params.TriggerType,
		userID:      params.UserID,
		funcName:    params.FunctionName,
		funcBrn:     params.FunctionBrn,
		funcVersion: params.FunctionVersion,
		fpath:       logpath,
		remain:      defaultUserLogLength,
		logbuf:      buf,
		logfile:     logfile,
	}
	r.waitg.Add(2)
	return r
}

func (s *kunLogStatStore) Receiver() string {
	return s.runtimeID
}

func (s *kunLogStatStore) String() string {
	return fmt.Sprintf("%s@%s", s.requestID, s.runtimeID)
}

func (s *kunLogStatStore) appendLogbuf(buf []byte, force bool) {
	if s.remain <= 0 && !force {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	logbuf := s.logbuf
	if logbuf == nil || (s.remain <= 0 && !force) {
		return
	}
	to := len(buf)
	if !force && to > s.remain {
		to = s.remain
	}
	if to <= 0 {
		return
	}
	logbuf.Write(buf[:to])
	s.remain -= to
	if to < len(buf) {
		if buf[to-1] != '\n' {
			logbuf.Write([]byte{'\n'})
			s.remain--
		}
	}
}

func (s *kunLogStatStore) WriteStdLog(from int, buf []byte, eof bool) (int, error) {
	if s.flags.Has(flagClosed) {
		return 0, errReceiverClosed
	}

	if eof {
		if from == StdoutLog {
			s.outLogDone()
		} else if from == StderrLog {
			s.errLogDone()
		}
	}
	if len(buf) == 0 {
		return 0, nil
	}

	s.appendLogbuf(buf, false)
	logfile := s.logfile
	if logfile == nil {
		return len(buf), nil
	}
	l := &userlog.UserLog{
		Created:        time.Now(),
		RequestID:      s.requestID,
		TriggerType:    s.triggerType,
		RuntimeID:      s.runtimeID,
		Source:         logSource[from],
		UserID:         s.userID,
		FuncName:       s.funcName,
		FunctionBrn:    s.funcBrn,
		Version:        s.funcVersion,
		InvocationTime: -1,
		MemoryUsage:    -1,
		ResponseStatus: -1,
		Mode:           "",
	}
	return logfile.Write(l, buf)
}

func (s *kunLogStatStore) WriteFunctionLog(log string) error {
	if s.flags.Has(flagClosed) {
		return errReceiverClosed
	}
	s.appendLogbuf([]byte(log), true)
	logfile := s.logfile
	if logfile == nil {
		return nil
	}
	l := &userlog.UserLog{
		Created:        time.Now(),
		TriggerType:    s.triggerType,
		RuntimeID:      s.runtimeID,
		RequestID:      s.requestID,
		Source:         logSource[OpenlessSysLog],
		Message:        []byte(log),
		UserID:         s.userID,
		FuncName:       s.funcName,
		FunctionBrn:    s.funcBrn,
		Version:        s.funcVersion,
		InvocationTime: -1,
		MemoryUsage:    -1,
		ResponseStatus: -1,
		Mode:           "",
	}
	_, err := logfile.Write(l, []byte(log))
	return err
}

type reportParameters struct {
	InvocationTime int64
	MemUsage       int64
	Mode           string
	Status         int
}

func (s *kunLogStatStore) WriteFunctionReportLog(log string, params *reportParameters) error {
	if s.flags.Has(flagClosed) {
		return errReceiverClosed
	}
	s.appendLogbuf([]byte(log), true)
	logfile := s.logfile
	if logfile == nil {
		return nil
	}
	l := &userlog.UserLog{
		Created:        time.Now(),
		TriggerType:    s.triggerType,
		RuntimeID:      s.runtimeID,
		RequestID:      s.requestID,
		Source:         logSource[OpenlessSysLog],
		Message:        []byte(log),
		UserID:         s.userID,
		FuncName:       s.funcName,
		FunctionBrn:    s.funcBrn,
		Version:        s.funcVersion,
		InvocationTime: params.InvocationTime,
		MemoryUsage:    params.MemUsage,
		Mode:           params.Mode,
		ResponseStatus: params.Status,
	}
	_, err := logfile.Write(l, []byte(log))
	return err
}

func (s *kunLogStatStore) SetMemUsed(used int64) {
	if s.maxMem < used {
		s.maxMem = used
	}
}

func (s *kunLogStatStore) LogFile() string {
	return s.fpath
}

func (s *kunLogStatStore) LogData() string {
	logbuf := s.logbuf
	if logbuf == nil {
		logs.V(4).Warnf("readlog %s after closed", s.String())
		return ""
	}
	data := logbuf.String()
	logbuf.Reset()
	logBufPool.Put(logbuf)
	s.logbuf = nil
	return data
}

func (s *kunLogStatStore) MemUsed() int64 {
	return s.maxMem
}

func (s *kunLogStatStore) setFlagLocked(flag bits) {
	if s.flags.Has(flag) {
		return
	}
	if flag == flagOutdone || flag == flagErrdone {
		s.waitg.Done()
	}
	s.flags.Set(flag)
}
func (s *kunLogStatStore) outLogDone() {
	if s.flags.Has(flagOutdone) {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.setFlagLocked(flagOutdone)
}

func (s *kunLogStatStore) errLogDone() {
	if s.flags.Has(flagErrdone) {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.setFlagLocked(flagErrdone)
}

func (s *kunLogStatStore) LogDone(set bool) bool {
	if !set {
		if !s.flags.Has(flagOutdone) {
			return false
		}
		if !s.flags.Has(flagErrdone) {
			return false
		}
		return true
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if set {
		s.setFlagLocked(flagErrdone)
		s.setFlagLocked(flagOutdone)
	}
	return true
}

func (s *kunLogStatStore) Close() (string, error) {
	if s.flags.Has(flagClosed) {
		return "", errReceiverClosed
	}
	s.mutex.Lock()
	if !s.flags.Has(flagClosed) {
		s.setFlagLocked(flagClosed)
		s.setFlagLocked(flagErrdone)
		s.setFlagLocked(flagOutdone)
	}
	data := s.LogData()
	s.mutex.Unlock()
	if s.logfile != nil {
		s.logfile.Close()
		s.logfile = nil
	}

	return data, nil
}

func (s *kunLogStatStore) Wait() {
	s.waitg.Wait()
}
