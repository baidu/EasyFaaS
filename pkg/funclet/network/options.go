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

import "github.com/spf13/pflag"

const (
	defaultMTU        = 1500
	defaultBridgeIP   = "172.33.0.1/24" // TODO: how to prevent network mask conflict with k8s
	defaultBridgeName = "miniBridge"
)

type NetworkOption struct {
	BridgeName string
	BridgeIP   string
	MTU        int

	EnableIcc bool
}

func NewNetworkOption() *NetworkOption {
	return &NetworkOption{
		BridgeName: defaultBridgeName,
		BridgeIP:   defaultBridgeIP,
		MTU:        defaultMTU,
	}
}
func (s *NetworkOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.BridgeName, "bridge-name", defaultBridgeName, "new bridge name")
	fs.StringVar(&s.BridgeIP, "bridge-ip", defaultBridgeIP, "new bridge ip")
	fs.IntVar(&s.MTU, "bridge-mtu", defaultMTU, "new bridge mtu")
}
