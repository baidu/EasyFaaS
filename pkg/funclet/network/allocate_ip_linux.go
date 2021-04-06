//+build linux

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
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/erikh/ping"
	"github.com/vishvananda/netlink"
)

// AllocateIP returns an unused IP for a specific process ID
// and saves it in the database.
func (c *ContainerNetwork) allocateIP() (ip net.IP, err error) {
	// Refresh the ipMap.
	ipMap, err := c.getIPMap()
	if err != nil {
		return nil, err
	}

	bridgeAddrs, _ := c.bridge.Addrs()
	ip = increaseIP(c.lastIP)

	for {
		switch {
		case !c.bridgeIPNet.Contains(ip):
			ip = c.bridgeIPNet.IP

		case func() bool {
			for _, addr := range bridgeAddrs {
				itfIP, _, _ := net.ParseCIDR(addr.String())
				if ip.Equal(itfIP) {
					return true
				}
			}
			return false
		}():

		// Skip broadcast ip
		case !isUnicastIP(ip, c.bridgeIPNet.Mask):

		case !func() bool { _, ok := ipMap[ip.String()]; return ok }():
			// use ICMP to check if the IP is in use, final sanity check.
			if !ping.Ping(&net.IPAddr{IP: ip, Zone: ""}, 5*time.Millisecond) {
				return ip, nil
			}
		}

		ip = increaseIP(ip)
		if ip.Equal(increaseIP(c.lastIP)) {
			break
		}
	}

	return nil, fmt.Errorf("could not find a suitable IP in network %s", c.bridgeIPNet.String())
}

func (c *ContainerNetwork) getIPMap() (map[string]struct{}, error) {
	// get the neighbors
	var (
		list []netlink.Neigh
		err  error
	)

	list, err = netlink.NeighList(c.bridge.Index, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve IPv4 neighbor information for interface %s: %v", c.bridge.Name, err)
	}

	ipMap := map[string]struct{}{}
	for _, entry := range list {
		ipMap[entry.String()] = struct{}{}
	}

	return ipMap, nil
}

func isUnicastIP(ip net.IP, mask net.IPMask) bool {
	// broadcast v4 ip
	if len(ip) == net.IPv4len && binary.BigEndian.Uint32(ip)&^binary.BigEndian.Uint32(mask) == ^binary.BigEndian.Uint32(mask) {
		return false
	}

	// global unicast
	return ip.IsGlobalUnicast()
}

// Increases IP address
func increaseIP(ip net.IP) net.IP {
	rawip := ipToBigInt(ip)
	rawip.Add(rawip, big.NewInt(1))
	return bigIntToIP(rawip)
}

// Converts a 4 bytes IP into a 128 bit integer
func ipToBigInt(ip net.IP) *big.Int {
	x := big.NewInt(0)
	return x.SetBytes(ip.To4())
}

// Converts 128 bit integer into a 4 bytes IP address
func bigIntToIP(v *big.Int) net.IP {
	return net.IP(v.Bytes())
}
