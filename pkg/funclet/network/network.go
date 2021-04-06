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
	"sync"
)

type NetworkManagerInterface interface {
	InitNetwork(opt *NetworkOption) error
	SetContainerNet(containerPid int) error
	UnsetContainerNet(containerPid int) error
}

type ContainerNetwork struct {
	bridge      *net.Interface
	bridgeIPNet *net.IPNet
	bridgeIP    net.IP

	lastIP net.IP

	vethPool []*Veth
	veth     []*Veth
	vethLock sync.Mutex
}

func NewNetworkManager() NetworkManagerInterface {
	return &ContainerNetwork{}
}

func (c *ContainerNetwork) InitNetwork(opt *NetworkOption) error {
	bridge, err := CreateBridge(opt)
	if err != nil {
		return fmt.Errorf("create bridge %s error %s", opt.BridgeName, err)
	}

	brNet, err := getInterfaceAddr(bridge.Name)
	if err != nil {
		return fmt.Errorf("retrieving IP/network of bridge %s failed: %v", bridge.Name, err)
	}

	ip, ipNet, err := net.ParseCIDR(brNet.String())
	if err != nil {
		return err
	}

	c.bridge = bridge
	c.bridgeIP = ip
	c.bridgeIPNet = ipNet
	c.lastIP = ip

	return c.disableIcc()
}

func (c *ContainerNetwork) SetContainerNet(containerPid int) error {
	veth, err := c.getNewVeth()
	if err != nil {
		return err
	}

	// bind veth to bridge
	if err := veth.Init(containerPid, c.bridge.Name); err != nil {
		return err
	}

	newIP := &net.IPNet{
		IP:   veth.IP,
		Mask: c.bridgeIPNet.Mask,
	}

	if err := veth.BindContainer(c.bridgeIP.String(), newIP); err != nil {
		return err
	}

	return nil
}

func (c *ContainerNetwork) getNewVeth() (*Veth, error) {
	c.vethLock.Lock()
	defer c.vethLock.Unlock()

	if len(c.vethPool) > 0 {
		newVeth := c.vethPool[0]
		c.veth = append(c.veth, newVeth)
		c.vethPool = c.vethPool[1:]

		return newVeth, nil
	}

	// create new veth
	lastIP, err := c.allocateIP()
	if err != nil {
		return nil, err
	}
	c.lastIP = lastIP

	newVeth := &Veth{
		ContainerPID: 0,
		IP:           lastIP,
	}
	c.veth = append(c.veth, newVeth)

	return newVeth, nil
}

func (c *ContainerNetwork) UnsetContainerNet(containerPid int) error {
	return c.deleteVeth(containerPid)
}

func (c *ContainerNetwork) deleteVeth(containerPid int) error {
	c.vethLock.Lock()
	defer c.vethLock.Unlock()

	for k, v := range c.veth {
		if v.ContainerPID == containerPid {
			v.ContainerPID = 0
			c.vethPool = append(c.vethPool, v)
			c.veth = append(c.veth[:k], c.veth[k+1:]...)

			return nil
		}
	}

	return fmt.Errorf("no container pid is %d", containerPid)
}
