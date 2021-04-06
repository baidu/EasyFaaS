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

package options

import (
	"fmt"
	"net"
	"strconv"

	"github.com/spf13/pflag"

	"github.com/baidu/openless/pkg/server"
)

type SecureServingOptions struct {
	BindAddress net.IP
	// BindPort is ignored when Listener is set, will serve https even with 0.
	BindPort int
	// BindNetwork is the type of network to bind to - defaults to "tcp", accepts "tcp",
	// "tcp4", and "tcp6".
	BindNetwork string

	// Listener is the secure server network listener.
	// either Listener or BindAddress/BindPort/BindNetwork is set,
	// if Listener is set, use it and omit BindAddress/BindPort/BindNetwork.
	Listener net.Listener
}

func NewSecureServingOptions() *SecureServingOptions {
	return &SecureServingOptions{
		BindAddress: net.ParseIP("0.0.0.0"),
		BindPort:    8080,
	}
}

func (s *SecureServingOptions) Validate() []error {
	if s == nil {
		return nil
	}

	errors := []error{}

	if s.BindPort < 0 || s.BindPort > 65535 {
		errors = append(errors, fmt.Errorf("--port %v must be between 0 and 65535, inclusive. 0 for turning off secure port", s.BindPort))
	}

	return errors
}

func (s *SecureServingOptions) AddFlags(fs *pflag.FlagSet) {
	if s == nil {
		return
	}

	fs.IPVar(&s.BindAddress, "bind-address", s.BindAddress, ""+
		"The IP address on which to listen for the --secure-port port. The "+
		"associated interface(s) must be reachable by the rest of the cluster, and by CLI/web "+
		"clients. If blank, all interfaces will be used (0.0.0.0 for all IPv4 interfaces and :: for all IPv6 interfaces).")
	fs.IntVar(&s.BindPort, "port", s.BindPort, ""+
		"The port on which to serve HTTP with authentication and authorization. If 0, "+
		"don't serve HTTP at all.")
}

// ApplyTo fills up serving information in the server configuration.
func (s *SecureServingOptions) ApplyTo(config **server.SecureServingInfo) error {
	if s == nil {
		return nil
	}
	if s.BindPort <= 0 && s.Listener == nil {
		return nil
	}

	if s.Listener == nil {
		var err error
		addr := net.JoinHostPort(s.BindAddress.String(), strconv.Itoa(s.BindPort))
		s.Listener, s.BindPort, err = CreateListener(s.BindNetwork, addr)
		if err != nil {
			return fmt.Errorf("failed to create listener: %v", err)
		}
	} else {
		if _, ok := s.Listener.Addr().(*net.TCPAddr); !ok {
			return fmt.Errorf("failed to parse ip and port from listener")
		}
		s.BindPort = s.Listener.Addr().(*net.TCPAddr).Port
		s.BindAddress = s.Listener.Addr().(*net.TCPAddr).IP
	}

	*config = &server.SecureServingInfo{
		Listener: s.Listener,
	}

	return nil
}

func CreateListener(network, addr string) (net.Listener, int, error) {
	if len(network) == 0 {
		network = "tcp"
	}
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to listen on %v: %v", addr, err)
	}

	// get port
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		ln.Close()
		return nil, 0, fmt.Errorf("invalid listen address: %q", ln.Addr().String())
	}

	return ln, tcpAddr.Port, nil
}
