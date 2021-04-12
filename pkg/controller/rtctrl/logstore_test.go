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
	"bytes"
	"io/ioutil"
	"testing"
	"time"
)

func TestBits(t *testing.T) {
	var b bits
	if b.Has(flagClosed) {
		t.Error(b)
		return
	}
	b.Set(flagClosed)
	if !b.Has(flagClosed) {
		t.Error(b)
		return
	}
	b.Set(flagOutdone)
	if !b.Has(flagClosed) {
		t.Error(b)
		return
	}
	if b != flagClosed|flagOutdone {
		t.Error(b)
		return
	}
}

func TestKunLogStatStore(t *testing.T) {
	params := &LogStatStoreParameter{
		RequestID:       "1111",
		RuntimeID:       "aaaa",
		UserID:          "user",
		FunctionName:    "func",
		FunctionVersion: "1",
		FilePath:        "/tmp",
		LogType:         "bos",
	}
	s := newLogStatStore(params).(*kunLogStatStore)
	if s.Receiver() != "aaaa" {
		t.Error(s.Receiver())
		return
	}
	if s.String() != "1111@aaaa" {
		t.Error(s.String())
		return
	}

	_, err := s.WriteStdLog(StdoutLog, []byte("stdout logline"), false)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = s.WriteStdLog(StderrLog, []byte("stderr logline"), false)
	if err != nil {
		t.Error(err)
		return
	}
	err = s.WriteFunctionLog("easyfaas logline")
	if err != nil {
		t.Error(err)
		return
	}
	if s.flags.Has(flagOutdone) || s.flags.Has(flagErrdone) {
		t.Error(s.flags)
		return
	}
	s.SetMemUsed(1024)
	if 1024 != s.MemUsed() {
		t.Errorf("memused: %d", s.MemUsed())
		return
	}

	s.WriteStdLog(StdoutLog, []byte("stdout logline 2"), true)
	s.WriteStdLog(StderrLog, []byte("stderr logline 2"), true)
	if !s.flags.Has(flagOutdone) || !s.flags.Has(flagErrdone) {
		t.Error(s.flags)
		return
	}
	logdata, err := s.Close()
	if err != nil {
		t.Error(err)
		return
	}
	if len(logdata) <= 0 {
		t.Error("no log?")
		return
	}
	t.Log(logdata)

	data, err := ioutil.ReadFile(s.LogFile())
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(data))
	ch := make(chan interface{})
	go func() {
		s.Wait()
		close(ch)
	}()
	select {
	case <-ch:
		// do nothing
	case <-time.After(1 * time.Second):
		t.Error("wait failed")
	}
}

type logStatStoreMock struct {
	memUsed int64
	logbuf  bytes.Buffer
	flags   bits
	from    int
}

func (m *logStatStoreMock) Receiver() string {
	return ""
}

func (m *logStatStoreMock) String() string {
	return ""
}

func (m *logStatStoreMock) WriteStdLog(from int, buf []byte, eof, lf bool) error {
	if m.from != from {
		return nil
	}
	m.logbuf.Write(buf)
	m.logbuf.WriteByte('\n')
	if eof {
		if from == StdoutLog {
			m.flags.Set(flagOutdone)
		} else if from == StderrLog {
			m.flags.Set(flagErrdone)
		}
	}

	return nil
}

func (m *logStatStoreMock) WriteFunctionLog(log string) error {
	m.logbuf.WriteString(log)
	return nil
}

func (m *logStatStoreMock) SetMemUsed(used int64) {
	m.memUsed = used
}

func (m *logStatStoreMock) LogData() []byte {
	return m.logbuf.Bytes()
}

func (m *logStatStoreMock) LogFile() string {
	return ""
}

func (m *logStatStoreMock) MemUsed() int64 {
	return m.memUsed
}

func (m *logStatStoreMock) Close() error {
	return nil
}

func (m *logStatStoreMock) LogDone(set bool) bool {
	return false
}

func (m *logStatStoreMock) Wait() {

}

type closeBuffer struct {
	buf bytes.Buffer
}

func (b *closeBuffer) Read(p []byte) (n int, err error) {
	return b.buf.Read(p)
}

func (b *closeBuffer) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

func (b *closeBuffer) WriteByte(t byte) error {
	return b.buf.WriteByte(t)
}

func (b *closeBuffer) Bytes() []byte {
	return b.buf.Bytes()
}

func (b *closeBuffer) Close() error {
	return nil
}

const letters = "abcdefghijklmnopqrstuvwxyz\n\000"
