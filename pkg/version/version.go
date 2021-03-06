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

package version

import (
	"fmt"
	"io"
	"runtime"
)

// Version contains versioning information.
// TODO: Add []string of api versions supported? It's still unclear
// how we'll want to distribute that information.
type Version struct {
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
func Get() *Version {
	return &Version{
		GitVersion:   gitVersion,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// PrintPretty xxx
func PrintPretty(w io.Writer) {
	fmt.Printf("GitVersion:   %s\n", gitVersion)
	fmt.Printf("GitCommit:    %s\n", gitCommit)
	fmt.Printf("GitTreeState: %s\n", gitTreeState)
	fmt.Printf("BuildDate:    %s\n", buildDate)
	fmt.Printf("GoVersion:    %s\n", runtime.Version())
	fmt.Printf("Compiler:     %s\n", runtime.Compiler)
	fmt.Printf("Platform:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// PrintStruct xxx
func PrintStruct(w io.Writer) {
	fmt.Printf("%#v\n", Get())
}
