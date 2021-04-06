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

package userlog

import (
	"bytes"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

type UserLogFile interface {
	Write(l *UserLog, buf []byte) (int, error)
	Close() error
}

// UserLog marshals encoded JSONLog objects
type UserLog struct {
	Created        time.Time `json:"ts"`
	RequestID      string    `json:"rid"`
	FunctionBrn    string    `json:"brn,omitempty"`
	TriggerType    string    `json:"triggerType,omitempty"`
	RuntimeID      string    `json:"runtimeID,omitempty"`
	FuncName       string    `json:"func,omitempty"`
	Version        string    `json:"version,omitempty"`
	UserID         string    `json:"userid,omitempty"`
	Source         string    `json:"src"`
	Message        []byte    `json:"msg"`
	InvocationTime int64     `json:"invocationTime,omitempty"`
	MemoryUsage    int64     `json:"memoryUsage,omitempty"`
	Mode           string    `json:"invokeMode, omitempty"`
	ResponseStatus int       `json:"responseStatus, omitempty"`
}

func (l *UserLog) Reset() {
	l.Created = time.Now()
	l.RequestID = ""
	l.TriggerType = ""
	l.RuntimeID = ""
	l.Source = ""
	l.FuncName = ""
	l.FunctionBrn = ""
	l.Version = ""
	l.UserID = ""
	l.Message = []byte{}
	l.InvocationTime = -1
	l.MemoryUsage = -1
	l.Mode = ""
	l.ResponseStatus = -1
}

// MarshalJSONBuf is an optimized JSON marshaller that avoids reflection
// and unnecessary allocation.
func (mj *UserLog) MarshalJSONBuf(buf *bytes.Buffer) error {
	var first = true

	buf.WriteString(`{`)
	if len(mj.RequestID) != 0 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"rid":`)
		ffjsonWriteJSONBytesAsString(buf, []byte(mj.RequestID))
	}
	if len(mj.Message) != 0 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"msg":`)
		ffjsonWriteJSONBytesAsString(buf, mj.Message)
	}
	if len(mj.Source) != 0 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"src":`)
		ffjsonWriteJSONBytesAsString(buf, []byte(mj.Source))
	}
	if len(mj.FunctionBrn) != 0 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"brn":`)
		ffjsonWriteJSONBytesAsString(buf, []byte(mj.FunctionBrn))
	}
	if len(mj.FuncName) != 0 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"func":`)
		ffjsonWriteJSONBytesAsString(buf, []byte(mj.FuncName))
	}
	if len(mj.Version) != 0 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"version":`)
		ffjsonWriteJSONBytesAsString(buf, []byte(mj.Version))
	}
	if len(mj.UserID) != 0 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"userid":`)
		ffjsonWriteJSONBytesAsString(buf, []byte(mj.UserID))
	}
	if len(mj.RuntimeID) != 0 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"runtimeID":`)
		ffjsonWriteJSONBytesAsString(buf, []byte(mj.RuntimeID))
	}
	if len(mj.TriggerType) != 0 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"triggerType":`)
		ffjsonWriteJSONBytesAsString(buf, []byte(mj.TriggerType))
	}
	if mj.InvocationTime != -1 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(fmt.Sprintf("\"invocationTime\":%d", mj.InvocationTime))
	}
	if mj.MemoryUsage != -1 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(fmt.Sprintf("\"memoryUsage\":%d", mj.MemoryUsage))
	}
	if len(mj.Mode) != 0 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"invokeMode":`)
		ffjsonWriteJSONBytesAsString(buf, []byte(mj.Mode))
	}
	if mj.ResponseStatus != -1 {
		if first {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(fmt.Sprintf("\"responseStatus\":%d", mj.ResponseStatus))
	}
	if !first {
		buf.WriteString(`,`)
	}

	created, err := fastTimeMarshalJSON(mj.Created)
	if err != nil {
		return err
	}

	buf.WriteString(`"ts":`)
	buf.WriteString(created)
	buf.WriteString("}\n")
	return nil
}

func ffjsonWriteJSONBytesAsString(buf *bytes.Buffer, s []byte) {
	const hex = "0123456789abcdef"

	buf.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' && b != '&' {
				i++
				continue
			}
			if start < i {
				buf.Write(s[start:i])
			}
			switch b {
			case '\\', '"':
				buf.WriteByte('\\')
				buf.WriteByte(b)
			case '\n':
				buf.WriteByte('\\')
				buf.WriteByte('n')
			case '\r':
				buf.WriteByte('\\')
				buf.WriteByte('r')
			default:
				buf.WriteString(`\u00`)
				buf.WriteByte(hex[b>>4])
				buf.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRune(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				buf.Write(s[start:i])
			}
			buf.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}

		if c == '\u2028' || c == '\u2029' {
			if start < i {
				buf.Write(s[start:i])
			}
			buf.WriteString(`\u202`)
			buf.WriteByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		buf.Write(s[start:])
	}
	buf.WriteByte('"')
}

const jsonFormat = `"` + time.RFC3339Nano + `"`

// fastTimeMarshalJSON avoids one of the extra allocations that
// time.MarshalJSON is making.
func fastTimeMarshalJSON(t time.Time) (string, error) {
	if y := t.Year(); y < 0 || y >= 10000 {
		// RFC 3339 is clear that years are 4 digits exactly.
		// See golang.org/issue/4556#c15 for more discussion.
		return "", errors.New("time.MarshalJSON: year outside of range [0,9999]")
	}
	return t.Format(jsonFormat), nil
}
