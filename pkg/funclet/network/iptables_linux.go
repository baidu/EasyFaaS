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

	"github.com/docker/libnetwork/iptables"
)

func (c *ContainerNetwork) disableIcc() error {
	var (
		table      = iptables.Filter
		chain      = "FORWARD"
		args       = []string{"-i", c.bridge.Name, "-o", c.bridge.Name, "-j"}
		acceptArgs = append(args, "ACCEPT")
		dropArgs   = append(args, "DROP")
	)

	iptables.Raw(append([]string{"-D", chain}, acceptArgs...)...)

	if !iptables.Exists(table, chain, dropArgs...) {
		if err := iptables.RawCombinedOutput(append([]string{"-A", chain}, dropArgs...)...); err != nil {
			return fmt.Errorf("unable to prevent intercontainer communication: %s", err.Error())
		}
	}

	return nil
}

// SetupNATOut adds NAT rules for outbound traffic with iptables.
func SetupNATOut(cidr string, action iptables.Action) error {
	masquerade := []string{
		"POSTROUTING", "-t", "nat",
		"-s", cidr,
		"-j", "MASQUERADE",
	}

	incl := append([]string{string(action)}, masquerade...)
	if _, err := iptables.Raw(
		append([]string{"-C"}, masquerade...)...,
	); err != nil || action == iptables.Delete {
		if output, err := iptables.Raw(incl...); err != nil {
			return err
		} else if len(output) > 0 {
			return &iptables.ChainError{
				Chain:  "POSTROUTING",
				Output: output,
			}
		}
	}

	return nil
}
