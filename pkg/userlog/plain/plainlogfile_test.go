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
	"fmt"
	"io/ioutil"
	"path"
	"testing"
	"time"

	"github.com/baidu/easyfaas/pkg/userlog"
)

func TestPlainLogfile(t *testing.T) {

	tmpdir, _ := ioutil.TempDir("", "jsonlog")
	tmpfile := path.Join(tmpdir, "testlog")
	jf, err := NewPlainLogFile(tmpfile, 1024)
	if err != nil {
		t.Error(err)
		return
	}
	l := &userlog.UserLog{
		Created:   time.Now(),
		RequestID: "66525001-1e97-469b-a151-cd264f519711",
		Source:    "faas",
	}
	msg := []byte("www.baidu.com log message")
	for i := 0; i < 5; i++ {
		_, err = jf.Write(l, msg)
		if err != nil {
			t.Error(err)
			return
		}
	}
	i := 0
	for ; i < 10; i++ {
		_, err = jf.Write(l, msg)
		if err != nil {
			t.Log(err)
			break
		}
	}
	if i < 10 {
		t.Error("capacity error")
		return
	}
	jf.Close()
	_, err = jf.Write(l, msg)
	if err == nil {
		t.Error("close failed")
		return
	}
	t.Log(err)

	data, err := ioutil.ReadFile(tmpfile)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Print(string(data))
}
