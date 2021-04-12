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
	"bufio"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/baidu/easyfaas/pkg/util/logs"
)

type DispatchServerV2 struct {
	config            *DispatcherV2Options
	runtimeDispatcher RuntimeDispatcher
	runtimeServer     *http.Server
	runnerServer      *http.Server

	storeMap storeMap
	statsMap statsMap
}

func NewDispatchServerV2(c *DispatcherV2Options, rtMap RuntimeDispatcher) *DispatchServerV2 {
	ds := &DispatchServerV2{
		config:            c,
		runtimeDispatcher: rtMap,
	}
	for _, item := range rtMap.RuntimeList() {
		ds.storeMap.setNXMap(item.RuntimeID)
	}
	return ds
}

func (s *DispatchServerV2) getRuntimeRouteHandler() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/invoke", s.invokeHandler)
	r.HandleFunc("/stdout", s.stdlogHandler(StdoutLog))
	r.HandleFunc("/stderr", s.stdlogHandler(StderrLog))
	r.HandleFunc("/statistic", s.statisticHandler)
	return r
}

func (s *DispatchServerV2) getRunnerRouteHandler() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/status", s.runnerHandler)
	return r
}

func (s *DispatchServerV2) ListenAndServe() {
	s.runtimeServer = s.serve(s.config.RuntimeServerAddress, s.getRuntimeRouteHandler())
	s.runnerServer = s.serve(s.config.RunnerServerAddress, s.getRunnerRouteHandler())
}

func (s *DispatchServerV2) serve(address string, router *mux.Router) *http.Server {
	ln, err := ListenerFromAddress(address, os.ModePerm)
	if err != nil {
		logs.Fatalf("listen %s error: %s", address, err.Error())
		panic(err)
	}

	serverMux := http.NewServeMux()
	serverMux.Handle("/", router)
	server := &http.Server{
		Handler: serverMux,
	}

	go func() {
		err := server.Serve(ln)
		if err != nil {
			logs.Fatalf("serve %s error: %s", address, err.Error())
		}
	}()
	return server
}

func setupHijackConn(w http.ResponseWriter, r *http.Request) (net.Conn, *bufio.ReadWriter, bool) {
	runtimeid := r.Header.Get("x-cfc-runtimeid")
	if len(runtimeid) == 0 {
		logs.V(4).Warnf("miss runtimeid %s", runtimeid)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return nil, nil, false
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		logs.V(4).Warnf("hijack failed %s", runtimeid)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return nil, nil, false
	}
	conn, buf, err := hijacker.Hijack()
	if err != nil {
		logs.V(4).Warnf("hijack %s connection error: %s", runtimeid, err.Error())
		http.Error(w, "invalid request", http.StatusBadRequest)
		return nil, nil, false
	}

	conn.Write([]byte{}) // set raw mode

	return conn, buf, true
}

func (s *DispatchServerV2) stdlogHandler(logfrom int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, buf, ok := setupHijackConn(w, r)
		if !ok {
			return
		}
		defer conn.Close()

		runtimeid := r.Header.Get("x-cfc-runtimeid")
		logs.Infof("stdlogHandler runtimeid %s logfrom %d connect", runtimeid, logfrom)
		defer logs.Infof("stdlogHandler runtimeid %s logfrom %d disconnect", runtimeid, logfrom)
		m, ok := s.storeMap.getMap(runtimeid)
		if !ok {
			logs.Errorf("stdlogHandler can't get map runtimeid %s", runtimeid)
			return
		}
		logDispatcher := m.NewStdLogDispatcher(runtimeid, logfrom, buf, 0)
		logDispatcher.Read()
	}
}

func (s *DispatchServerV2) statisticHandler(w http.ResponseWriter, r *http.Request) {
	conn, _, ok := setupHijackConn(w, r)
	if !ok {
		return
	}
	defer conn.Close()
	runtimeid := r.Header.Get("x-cfc-runtimeid")
	nowstat := &statsConn{
		name:    runtimeid,
		conn:    conn,
		memused: 0,
	}
	s.statsMap.set(runtimeid, nowstat)
	defer s.statsMap.del(runtimeid, nowstat)

	m, ok := s.storeMap.getMap(runtimeid)
	if !ok {
		logs.Errorf("statisticHandler can't get map runtimeid %s", runtimeid)
	}
	receiver := &statinfoReceiver{
		name: runtimeid,
		conn: conn,
	}
	receiver.Recv(func(info *statInfo) error {
		if m != nil {
			store, err := m.getLast()
			if err != nil {
				logs.Errorf("statisticHandler map %s can't get last request %s", m.String(), runtimeid)
			}
			if store != nil {
				store.SetMemUsed(info.MemUsed)
			}
		}
		nowstat.memused = info.MemUsed
		return nil
	})
}

func (s *DispatchServerV2) invokeHandler(w http.ResponseWriter, r *http.Request) {
	conn, _, ok := setupHijackConn(w, r)
	if !ok {
		logs.Error("setupHijackConn failed")
		return
	}
	defer conn.Close()

	runtimeid := r.Header.Get("x-cfc-runtimeid")
	commitid := r.Header.Get("x-cfc-commitid")
	// TODO: [improve] hostip here is always 127.0.0.1
	hostip := r.Header.Get("x-cfc-hostip")
	logs.V(6).Infof("runtime %s@%s connect", runtimeid, hostip)

	runtime, err := s.runtimeDispatcher.GetRuntime(runtimeid)
	if err != nil {
		logs.V(3).Errorf("runtime %s not found", runtimeid)
		return
	}

	warmNotify := make(chan struct{})
	params := &startRuntimeParams{
		commitID:   commitid,
		hostIP:     hostip,
		conn:       conn,
		warmNotify: warmNotify,
		urlParams:  r.URL.Query(),
	}
	go func() {
		select {
		case <-warmNotify:
			logs.V(9).Infof("[resource modify]-[increase]: runtime %s, resource %s", runtime.RuntimeID, runtime.Resource)
			s.runtimeDispatcher.IncreaseUsedResource(runtime.Resource)
		}
	}()
	if err := runtime.startRuntimeLoop(params); err != nil {
		logs.Errorf("init runtime %s failed: %s", runtime.RuntimeID, err.Error())
		return
	}
}

func (s *DispatchServerV2) runnerHandler(w http.ResponseWriter, r *http.Request) {
	runtimeID := r.Header.Get("x-cfc-runtimeid")
	logs.Infof("runner %s connect", runtimeID)

	conn, buf, ok := setupHijackConn(w, r)
	if !ok {
		logs.Errorf("setupHijackConn runtime %s failed", runtimeID)
		return
	}

	rt, err := s.runtimeDispatcher.GetRuntime(runtimeID)
	if err != nil {
		logs.Errorf("can't get runtime %s from runner connection", runtimeID)
		return
	}
	if err := rt.startRunnerLoop(&startRunnerParams{conn: conn, buf: buf}); err != nil {
		logs.Errorf("init runner %s failed: %s", rt.RuntimeID, err.Error())
	}
}

func (s *DispatchServerV2) StartRecvLog(runtimeID, requestID string, store LogStatStore) {
	st := s.statsMap.get(runtimeID)
	if st != nil {
		store.SetMemUsed(st.memused)
	}
	s.storeMap.set(runtimeID, requestID, store)
}

func (s *DispatchServerV2) StopRecvLog(runtimeID, requestID string, store LogStatStore) {
	s.storeMap.del(runtimeID, requestID, store)
}
