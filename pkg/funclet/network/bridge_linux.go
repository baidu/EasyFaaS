// +build linux

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

package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/docker/libnetwork/iptables"
	"github.com/vishvananda/netlink"

	"github.com/baidu/openless/pkg/util/logs"
)

func CreateBridge(opt *NetworkOption) (*net.Interface, error) {
	// Validate the options.
	if len(opt.BridgeIP) < 1 {
		return nil, fmt.Errorf("network bridge ip address can not be empty")
	}
	if len(opt.BridgeName) < 1 {
		return nil, fmt.Errorf("network bridge name can not be empty")
	}

	// Set the defaults.
	if opt.MTU < 1 {
		opt.MTU = defaultMTU
	}

	bridge, err := net.InterfaceByName(opt.BridgeName)
	if err == nil {
		// Bridge already exists, return early.
		return bridge, nil
	}

	if !strings.Contains(err.Error(), "no such network interface") {
		return nil, fmt.Errorf("getting interface %s failed: %v", opt.BridgeName, err)
	}

	// Create *netlink.Bridge object.
	logs.Infof("start create bridge %s(%s)", opt.BridgeName, opt.BridgeIP)
	la := netlink.NewLinkAttrs()
	la.Name = opt.BridgeName
	la.MTU = opt.MTU
	br := &netlink.Bridge{LinkAttrs: la}
	if err := netlink.LinkAdd(br); err != nil {
		return nil, fmt.Errorf("bridge creation for %s failed: %v", opt.BridgeName, err)
	}

	// Setup ip address for bridge.
	addr, err := netlink.ParseAddr(opt.BridgeIP)
	if err != nil {
		return nil, fmt.Errorf("parsing address %s failed: %v", opt.BridgeIP, err)
	}
	if err := netlink.AddrAdd(br, addr); err != nil {
		return nil, fmt.Errorf("adding address %s to bridge %s failed: %v", addr.String(), opt.BridgeName, err)
	}

	// Validate that the IPAddress is there!
	if _, err := getInterfaceAddr(opt.BridgeName); err != nil {
		return nil, err
	}

	// Add NAT rules for iptables.
	if err := SetupNATOut(opt.BridgeIP, iptables.Insert); err != nil {
		return nil, fmt.Errorf("setting up NAT outbound for %s failed: %v", opt.BridgeName, err)
	}

	// Bring the bridge up.
	if err := netlink.LinkSetUp(br); err != nil {
		return nil, fmt.Errorf("bringing bridge %s up failed: %v", opt.BridgeName, err)
	}

	return net.InterfaceByName(opt.BridgeName)
}

// GetInterfaceAddr returns the IPv4 address of a network interface.
func getInterfaceAddr(name string) (*net.IPNet, error) {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("getting interface %s failed: %v", name, err)
	}

	addrs, err := netlink.AddrList(iface, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("listings addresses for %s failed: %v", name, err)
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("interface %s has no IP addresses", name)
	}

	return addrs[0].IPNet, nil
}
