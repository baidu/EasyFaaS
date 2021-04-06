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
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
)

func filterNetAddr(name, mask string, bits int) (net.IP, error) {
	mip := &net.IPNet{
		IP:   net.ParseIP(mask),
		Mask: net.CIDRMask(bits, 32),
	}
	netis, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, n := range netis {
		if n.Name == name {
			addrs, err := n.Addrs()
			if err != nil {
				return nil, err
			}
			for _, a := range addrs {
				ipnet := a.(*net.IPNet)
				if ipnet.IP.IsGlobalUnicast() &&
					len(ipnet.IP) == net.IPv4len &&
					mip.Contains(ipnet.IP) {
					return ipnet.IP, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("interface %s does not have a specific address in network %s/%d", name, mask, bits)
}

func ListenerFromAddress(addr string, fileMode os.FileMode) (net.Listener, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	if err = TryConnectAddress(addr); err != nil {
		return nil, err
	}

	if u.Scheme == "tcp" {
		query := u.Query()
		if name := query.Get("interface"); name != "" {
			maskbits, err := strconv.Atoi(query.Get("maskbits"))
			if err != nil {
				maskbits = 0
			}
			ip, err := filterNetAddr(name, u.Hostname(), maskbits)
			if err != nil {
				return nil, err
			}
			port := u.Port()
			addr := ip.String()
			if port != "" {
				addr = fmt.Sprintf("%s:%s", addr, port)
			}
			return net.Listen(u.Scheme, addr)
		}
		return net.Listen(u.Scheme, u.Host)
	}
	if u.Scheme == "unix" {
		ln, err := net.Listen(u.Scheme, u.Path)
		if err != nil {
			return nil, err
		}
		// os.FileMode(0755) is default
		os.Chmod(u.Path, fileMode)
		return ln, nil
	}
	return nil, errors.New("invalid address schema")
}

func TryConnectAddress(addr string) error {
	u, err := url.Parse(addr)
	if err != nil {
		return err
	}
	if u.Scheme == "unix" {
		_, err := os.Stat(u.Path)
		if err != nil && os.IsNotExist(err) {
			return nil
		}
		conn, err := net.Dial("unix", u.Path)
		if err != nil {
			os.Remove(u.Path)
			return nil
		}
		conn.Close()
		return fmt.Errorf("%s in use", u.Path)
	}
	if u.Scheme == "tcp" {
		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			return err
		}
		addr := host
		if host == "0.0.0.0" {
			addr = "127.0.0.1"
		}
		if port != "" {
			addr = fmt.Sprintf("%s:%s", addr, port)
		}
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return nil
		}
		conn.Close()
		return fmt.Errorf("%s in use", u.Host)
	}
	return errors.New("invalid address schema")
}
