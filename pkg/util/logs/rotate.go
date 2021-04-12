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

package logs

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/baidu/easyfaas/pkg/util/bytefmt"
)

var (
	pid      = os.Getpid()
	host     = "unknownhost"
	userName = "unknownuser"
)

func init() {
	h, err := os.Hostname()
	if err == nil {
		host = shortHostname(h)
	}

	current, err := user.Current()
	if err == nil {
		userName = current.Username
	}

	// Sanitize userName since it may contain filepath separators on Windows.
	userName = strings.Replace(userName, `\`, "_", -1)
}

// Rotate implement a lock-free logrotater
type Rotate struct {
	dir   string
	name  string
	size  int64
	tag   string
	ofile *os.File // old file
	file  *os.File // current use file
	fname string
}

// NewRotate create a Rotate
func NewRotate(dir, name, tag string, opts ...RotateOpt) *Rotate {
	f, fname, err := create(dir, tag, time.Now())
	if err != nil {
		return nil
	}
	r := &Rotate{
		dir:   dir,
		name:  name,
		size:  2 * bytefmt.Gigabyte,
		tag:   tag,
		file:  f,
		fname: fname,
	}
	for _, opt := range opts {
		opt(r)
	}
	go r.rotate()
	return r
}

// Write implement interface of io.Writer
func (r *Rotate) Write(p []byte) (int, error) {
	return r.file.Write(p)
}

func (r *Rotate) rotate() {
	for {
		if r.ofile != nil {
			r.ofile.Close()
			r.ofile = nil
		}
		fileinfo, err := os.Stat(r.fname)
		if err != nil {
			fmt.Println("file info error")
			return
		}
		if fileinfo.Size() > r.size {
			f, fname, err := create(r.dir, r.tag, time.Now())
			if err != nil {
				fmt.Printf("create file error, %s", err.Error())
				return
			}
			r.ofile = r.file
			r.file = f
			r.fname = fname
		}
		time.Sleep(5 * time.Second)
	}
}

// shortHostname returns its argument, truncating at the first period.
// For instance, given "www.google.com" it returns "www".
func shortHostname(hostname string) string {
	if i := strings.Index(hostname, "."); i >= 0 {
		return hostname[:i]
	}
	return hostname
}

// logName returns a new log file name containing tag, with start time t, and
// the name for the symlink for tag.
func logName(tag string, t time.Time) (name, link string) {
	name = fmt.Sprintf("%s.%s.%s.log.%s.%04d%02d%02d-%02d%02d%02d.%d",
		program,
		host,
		userName,
		tag,
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		pid)
	return name, program + "." + tag
}

// create creates a new log file and returns the file and its filename, which
// contains tag ("INFO", "FATAL", etc.) and t.  If the file is created
// successfully, create also attempts to update the symlink for that tag, ignoring
// errors.
func create(dir, tag string, t time.Time) (f *os.File, filename string, err error) {
	name, link := logName(tag, t)
	fname := filepath.Join(dir, name)
	f, err = os.Create(fname)
	if err == nil {
		symlink := filepath.Join(dir, link)
		os.Remove(symlink)        // ignore err
		os.Symlink(name, symlink) // ignore err
		return f, fname, nil
	}
	return nil, "", fmt.Errorf("log: cannot create log: %v", err)
}

// RotateOpt is options
type RotateOpt func(*Rotate)

// SetRotateSize set file rotate max size
func SetRotateSize(size int64) RotateOpt {
	return func(r *Rotate) {
		r.size = size
	}
}
