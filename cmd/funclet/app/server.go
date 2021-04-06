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

package app

import (
	"net"
	"os"
	"syscall"

	"github.com/baidu/openless/cmd/funclet/options"
	"github.com/baidu/openless/pkg/funclet"
	genericserver "github.com/baidu/openless/pkg/server"
	"github.com/baidu/openless/pkg/util/logs"
	"github.com/baidu/openless/pkg/version"
)

// Run runs the specified Funclet Server with the given Dependencies.
// This should never exit.
func Run(runOptions *options.FuncletOptions, stopCh <-chan struct{}) error {
	logs.Infof("Version: %+v", version.Get())

	s, err := CreateUnixServerChain(runOptions, stopCh)
	if err != nil {
		return err
	}

	return s.PrepareRun().Run(stopCh)
}

func CreateUnixServerChain(runOptions *options.FuncletOptions, stopCh <-chan struct{}) (*genericserver.GenericServer, error) {
	config := genericserver.NewRecommendedConfig()
	if err := runOptions.RecommendedOptions.ApplyTo(config); err != nil {
		return nil, err
	}
	s, err := config.Complete().New("minifunclet")
	if err != nil {
		return nil, err
	}

	// funclet needs to prepare the directory for user containers
	// lift the restrictions of the new file's permissions
	syscall.Umask(0)
	f, err := funclet.InitFunclet(runOptions, stopCh)
	if err != nil {
		return nil, err
	}
	socksPath := runOptions.FuncletApiSocks
	if err := os.RemoveAll(socksPath); err != nil {
		return nil, err
	}
	ln, err := net.Listen("unix", socksPath)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(socksPath, os.ModePerm); err != nil {
		return nil, err
	}
	config.SecureServingInfo = &genericserver.SecureServingInfo{
		Listener: ln,
	}
	s, err = config.Complete().New("funclet_unix")
	if err != nil {
		return nil, err
	}
	f.InstallAPI(s.Handler.GoRestfulContainer)
	return s, nil
}
