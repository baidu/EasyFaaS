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
	"github.com/spf13/pflag"

	"github.com/baidu/easyfaas/pkg/server"
)

// ServerRunOptions contains the options while running a generic api server.
type ServerRunOptions struct {
	MaxRequestsInFlight int
	Version             bool
}

// NewServerRunOptions create a default options for a generic api server.
func NewServerRunOptions() *ServerRunOptions {
	defaults := server.NewConfig()

	return &ServerRunOptions{
		MaxRequestsInFlight: defaults.MaxRequestsInFlight,
	}
}

// ApplyTo applies the run options to the method receiver and returns self
func (s *ServerRunOptions) ApplyTo(c *server.Config) error {
	c.MaxRequestsInFlight = s.MaxRequestsInFlight

	return nil
}

// AddUniversalFlags parse the flags
func (s *ServerRunOptions) AddUniversalFlags(fs *pflag.FlagSet) {
	// Note: the weird ""+ in below lines seems to be the only way to get gofmt to
	// arrange these text blocks sensibly. Grrr.

	fs.IntVar(&s.MaxRequestsInFlight, "max-requests-inflight", s.MaxRequestsInFlight, ""+
		"The maximum number of non-mutating requests in flight at a given time. When the server exceeds this, "+
		"it rejects requests. Zero for no limit.")
}
