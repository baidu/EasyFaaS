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

package client

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/baidu/openless/pkg/api"
	"github.com/baidu/openless/pkg/util/flag"

	"github.com/gorilla/mux"
)

const ServerHost = "http://127.0.0.1:7777"
const OtherServerHost = "http://127.0.0.1:7788"
const ErrorServerHost = "http:/%%a.openless.com/just/a/path"

func TestNewControllerClient(t *testing.T) {
	ds := []struct {
		host string
		err  error
	}{
		{host: ServerHost, err: nil},
		{host: ErrorServerHost, err: InvaildConfigError{Reason: fmt.Sprintf("invaild host %s", ErrorServerHost)}},
	}
	for _, item := range ds {
		pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
		opt := NewControllerClientOptions()
		opt.AddFlags(pflag.CommandLine)
		pflag.Set("controller-host", item.host)
		flag.InitFlags()
		fmt.Println(opt.Host)
		_, err := NewControllerClient(opt)
		t.Logf("err = %+v", err)
		if item.err != err {
			t.Errorf("new controller client expect err %+v, but got %+v", item.err, err)
		}
	}
}

func TestInvoke(t *testing.T) {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	opt := NewControllerClientOptions()
	opt.AddFlags(pflag.CommandLine)
	pflag.Set("controller-host", ServerHost)
	flag.InitFlags()
	client, err := NewControllerClient(opt)
	if err != nil {
		t.Errorf("new client with err %s", err.Error())
		return
	}

	ds := []struct {
		host      string
		withError bool
		err       error
	}{
		{host: ServerHost, withError: false, err: nil},
		{host: ServerHost, withError: true, err: nil},
		{host: OtherServerHost, withError: true, err: errors.New(fmt.Sprintf("dial tcp4 127.0.0.1:7777: connect: connection refused"))},
	}
	for _, item := range ds {
		router := newControllerClientRouter(item.withError)
		ts := httptest.NewUnstartedServer(router)
		u, _ := url.Parse(item.host)
		l, err := net.Listen("tcp", u.Host)
		if err != nil {
			t.Errorf("test server listen err: %+v", err)
			return
		}
		ts.Listener = l
		ts.Start()
		bodyStr := "{\"k1\":\"v1\"}"
		ir := &api.InvokeRequest{
			UserID:      "df391b08c64c426a81645468c75163a5",
			FunctionBRN: "brn:cloud:faas:bj:cd64f99c69d7c404b61de0a4f1865834:function:concurrentHello:1",
			Body:        &bodyStr,
			RequestID:   "xxx",
		}
		res, err := client.Invoke(ir)
		t.Logf("res = %v", res)
		t.Logf("err = %v", err)
		if err != item.err {
			ts.Close()
			if err != nil && item.err == nil {
				t.Errorf("controller invoke expect err nil, but got %+v", err)
			} else if item.err != nil && err == nil {
				t.Errorf("controller invoke expect err %+v, but got nil", item.err)
			} else if item.err.Error() != err.Error() {
				t.Errorf("controller invoke expect err %+v, but got %+v", item.err, err)
			} else {
				continue
			}
			return
		}
		ts.Close()
	}
	return
}

func newControllerClientRouter(withError bool) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/v1/functions/{functionName}/invocations", InvokeHandler(withError)).Methods("POST")
	return router
}

func InvokeHandler(withError bool) func(w http.ResponseWriter, r *http.Request) {
	if withError {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("oops"))
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	}
}
