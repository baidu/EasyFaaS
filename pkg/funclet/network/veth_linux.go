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
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/baidu/openless/pkg/util/logs"
)

func (v *Veth) Init(pid int, bridgeName string) error {
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("getting link %s failed: %v", bridgeName, err)
	}

	la := netlink.NewLinkAttrs()
	la.Name = fmt.Sprintf("%s-%d", "miniv", pid)
	la.MasterIndex = br.Attrs().Index

	v.vethPair = &netlink.Veth{
		LinkAttrs: la,
		PeerName:  fmt.Sprintf("ethc%d", pid),
	}
	v.ContainerPID = pid

	if err := netlink.LinkAdd(v.vethPair); err != nil {
		return fmt.Errorf("create veth pair named [ %#v ] failed: %v", v.vethPair, err)
	}

	return nil
}

func (v *Veth) BindContainer(gatewayIP string, addr *net.IPNet) error {
	logs.Infof("start bind container(pid: %d) with veth(ip: %s)", v.ContainerPID, addr.IP.String())

	// Get the peer link.
	peer, err := netlink.LinkByName(v.vethPair.PeerName)
	if err != nil {
		return fmt.Errorf("getting peer interface %s failed: %v", v.vethPair.PeerName, err)
	}

	// Put peer interface into the network namespace of specified PID.
	if err := netlink.LinkSetNsPid(peer, v.ContainerPID); err != nil {
		return fmt.Errorf("adding peer interface to network namespace of pid %d failed: %v", v.ContainerPID, err)
	}

	// Bring the veth pair up.
	if err := netlink.LinkSetUp(v.vethPair); err != nil {
		return fmt.Errorf("bringing local veth pair [ %#v ] up failed: %v", peer, err)
	}

	// Lock the OS Thread so we don't accidentally switch namespaces.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origns, err := netns.Get()
	if err != nil {
		return fmt.Errorf("getting current network namespace failed: %v", err)
	}
	defer origns.Close()

	newns, err := netns.GetFromPid(v.ContainerPID)
	if err != nil {
		return fmt.Errorf("getting network namespace for pid %d failed: %v", v.ContainerPID, err)
	}
	defer newns.Close()

	if err := netns.Set(newns); err != nil {
		return fmt.Errorf("entering network namespace failed: %v", err)
	}

	if err := netlink.LinkSetDown(peer); err != nil {
		return fmt.Errorf("bringing interface [ %#v ] down failed: %v", peer, err)
	}

	if err := netlink.LinkSetName(peer, "eth0"); err != nil {
		return fmt.Errorf("renaming interface %s to %s failed: %v", v.vethPair.PeerName, "eth0", err)
	}

	// Add the IP address.
	ipAddr := &netlink.Addr{IPNet: addr, Label: ""}
	if err := netlink.AddrAdd(peer, ipAddr); err != nil {
		return fmt.Errorf("setting %s interface ip to %s failed: %v", v.vethPair.PeerName, addr.String(), err)
	}

	if err := netlink.LinkSetUp(peer); err != nil {
		return fmt.Errorf("bringing interface [ %#v ] up failed: %v", peer, err)
	}

	gw := net.ParseIP(gatewayIP)
	err = netlink.RouteAdd(&netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: peer.Attrs().Index,
		Gw:        gw,
	})
	if err != nil {
		return fmt.Errorf("adding route %s to interface %s failed: %v", gw.String(), v.vethPair.PeerName, err)
	}

	if err := netns.Set(origns); err != nil {
		return fmt.Errorf("switching back to original namespace failed: %v", err)
	}

	return nil
}
