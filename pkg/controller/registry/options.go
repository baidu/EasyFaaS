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

package registry

import (
	"fmt"

	"github.com/spf13/pflag"
)

const (
	NOAuthType  = "noauth"
	BCEAuthType = "cloud"
)

type Options struct {
	Endpoint     string
	Version      string
	AuthTransfer bool
	Auth         *AuthOption
}

type AuthOption struct {
	Name     string
	Params   map[string]interface{}
	ParamStr string
}

func NewOption() *Options {
	return &Options{
		Endpoint:     "http://127.0.0.1:8080",
		Version:      "v1",
		AuthTransfer: false,
		Auth: &AuthOption{
			Name:     NOAuthType,
			ParamStr: "",
		},
	}
}

func NewEmptyOption() *Options {
	return &Options{
		Endpoint:     "",
		Version:      "",
		AuthTransfer: false,
		Auth:         &AuthOption{},
	}
}
func (o *Options) IsEmpty() bool {
	return o.Endpoint == ""
}
func (s *Options) AddFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&s.Endpoint, getFlagName(prefix, "repository-endpoint"), s.Endpoint, "function repository endpoint")
	fs.StringVar(&s.Version, getFlagName(prefix, "repository-version"), s.Version, "function repository api version")
	fs.BoolVar(&s.AuthTransfer, getFlagName(prefix, "repository-auth-transfer"), s.AuthTransfer, "transfer authorization to function repository")
	fs.StringVar(&s.Auth.Name, getFlagName(prefix, "repository-auth-type"), s.Auth.Name, "function repository auth type")
	fs.StringVar(&s.Auth.ParamStr, getFlagName(prefix, "repository-auth-params"), s.Auth.ParamStr, "function repository auth params")
}

func getFlagName(prefix, flagName string) string {
	if prefix == "" {
		return flagName
	}
	return fmt.Sprintf("%s-%s", prefix, flagName)
}
