//+build !linux

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
	"math/big"
	"net"
)

// AllocateIP returns an unused IP for a specific process ID
// and saves it in the database.
func (c *ContainerNetwork) allocateIP() (ip net.IP, err error) {
	return nil, nil
}

func (c *ContainerNetwork) getIPMap() (map[string]struct{}, error) {
	return nil, nil
}

func isUnicastIP(ip net.IP, mask net.IPMask) bool {
	return false
}

// Increases IP address
func increaseIP(ip net.IP) net.IP {
	return nil
}

// Converts a 4 bytes IP into a 128 bit integer
func ipToBigInt(ip net.IP) *big.Int {
	return nil
}

// Converts 128 bit integer into a 4 bytes IP address
func bigIntToIP(v *big.Int) net.IP {
	return nil
}
