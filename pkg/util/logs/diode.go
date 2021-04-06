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

// Provides a thread-safe, lock-free, non-blocking io.Writer wrapper.
package logs

import (
	"context"
	"io"
	"sync"
	"time"

	"code.cloudfoundry.org/go-diodes"
)

var bufPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 2000)
	},
}

type Alerter func(missed int)

// DiodeWriter is a io.Writer wrapper that uses a diode to make Write lock-free,
// non-blocking and thread safe.
type DiodeWriter struct {
	w    io.Writer
	d    *diodes.ManyToOne
	p    *diodes.Poller
	c    context.CancelFunc
	done chan struct{}
}

// NewDiodeWriter creates a writer wrapping w with a many-to-one diode in order to
// never block log producers and drop events if the writer can't keep up with
// the flow of data.
//
// Use a diode.Writer when
//
//     wr := diode.NewDiodeWriter(w, 1000, 10 * time.Millisecond, func(missed int) {
//         log.Printf("Dropped %d messages", missed)
//     })
//     log := zerolog.New(wr)
//
//
// See code.cloudfoundry.org/go-diodes for more info on diode.
func NewDiodeWriter(w io.Writer, size int, poolInterval time.Duration, f Alerter) DiodeWriter {
	ctx, cancel := context.WithCancel(context.Background())
	d := diodes.NewManyToOne(size, diodes.AlertFunc(f))
	dw := DiodeWriter{
		w: w,
		d: d,
		p: diodes.NewPoller(d,
			diodes.WithPollingInterval(poolInterval),
			diodes.WithPollingContext(ctx)),
		c:    cancel,
		done: make(chan struct{}),
	}
	go dw.poll()
	return dw
}

func (dw DiodeWriter) Write(p []byte) (n int, err error) {
	// p is pooled in zerolog so we can't hold it passed this call, hence the
	// copy.
	p = append(bufPool.Get().([]byte), p...)
	dw.d.Set(diodes.GenericDataType(&p))
	return len(p), nil
}

// Close releases the diode poller and call Close on the wrapped writer if
// io.Closer is implemented.
func (dw DiodeWriter) Close() error {
	dw.c()
	<-dw.done
	if w, ok := dw.w.(io.Closer); ok {
		return w.Close()
	}
	return nil
}

func (dw DiodeWriter) poll() {
	defer close(dw.done)
	for {
		d := dw.p.Next()
		if d == nil {
			return
		}
		p := *(*[]byte)(d)
		dw.w.Write(p)
		bufPool.Put(p[:0])
	}
}
