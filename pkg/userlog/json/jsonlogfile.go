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

package json

import (
	"bytes"
	"errors"
	"os"
	"sync/atomic"

	"github.com/baidu/easyfaas/pkg/userlog"
)

var (
	LineByte        = []byte{'\n'}
	errLogClosed    = errors.New("log file closed")
	errOverCapacity = errors.New("log file over capacity")
)

func init() {
	userlog.RegisterLogWriter("json", NewJSONLogFile)
}

type JSONLogFile struct {
	writer   *os.File
	capacity int32
}

func NewJSONLogFile(fpath string, cap int) (userlog.UserLogFile, error) {
	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	w := &JSONLogFile{
		writer:   f,
		capacity: int32(cap),
	}
	return w, nil
}

func (f *JSONLogFile) Write(l *userlog.UserLog, buf []byte) (int, error) {
	w := f.writer
	if w == nil {
		return 0, errLogClosed
	}
	if f.capacity <= 0 {
		return 0, errOverCapacity
	}
	tmp := bytes.NewBuffer(nil)
	i, j := 0, 0
	for i < len(buf) {
		j = bytes.Index(buf[i:], LineByte)
		if j < 0 {
			j = len(buf)
		} else if j > 0 {
			j = i + j
		} else {
			i++
			continue
		}
		l.Message = buf[i:j]
		if err := l.MarshalJSONBuf(tmp); err != nil {
			return 0, err
		}
		i = j + 1
	}
	w.Write(tmp.Bytes())
	atomic.AddInt32(&f.capacity, int32(-i))
	return i, nil
}

func (f *JSONLogFile) Close() error {
	w := f.writer
	f.writer = nil
	return w.Close()
}
