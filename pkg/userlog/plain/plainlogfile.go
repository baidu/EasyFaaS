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

// Package plain
package plain

import (
	"bytes"
	"errors"
	"os"
	"sync/atomic"
	"time"

	"github.com/baidu/easyfaas/pkg/userlog"
)

func init() {
	userlog.RegisterLogWriter("plain", NewPlainLogFile)
	userlog.RegisterLogWriter("bos", NewPlainLogFile)
}

type plainLogFile struct {
	writer   *os.File
	capacity int32
}

func NewPlainLogFile(fpath string, cap int) (userlog.UserLogFile, error) {
	file, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &plainLogFile{
		writer:   file,
		capacity: int32(cap),
	}, nil
}

var (
	LineByte        = []byte{'\n'}
	errLogClosed    = errors.New("log file closed")
	errOverCapacity = errors.New("log file over capacity")
)

func (f *plainLogFile) Write(l *userlog.UserLog, buf []byte) (int, error) {
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
		tmp.WriteString(l.Created.UTC().Format(time.RFC3339))
		tmp.WriteByte('\t')
		tmp.Write(buf[i:j])
		tmp.WriteByte('\n')
		i = j + 1
	}
	w.Write(tmp.Bytes())
	atomic.AddInt32(&f.capacity, int32(-i))
	return i, nil
}

func (f *plainLogFile) Close() error {
	w := f.writer
	f.writer = nil
	return w.Close()
}
