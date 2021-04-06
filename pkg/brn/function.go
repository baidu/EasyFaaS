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

package brn

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/baidu/openless/pkg/api"
)

var (
	RegAlias      = regexp.MustCompile("^[a-zA-Z0-9-_]+$")
	regBrn        = regexp.MustCompile(`^(brn:(cloud[a-zA-Z-]*):faas:)([a-z]{2,5}[0-9]*:)([0-9a-z]{32}:)(function:)([a-zA-Z0-9-_]+)(:(\$LATEST|[a-zA-Z0-9-_]+))?$`)
	regPartialBrn = regexp.MustCompile(`^([0-9a-z]{32}:)?([a-zA-Z0-9-_]+)(:(\$LATEST|[a-zA-Z0-9-_]+))?$`)

	RegNotMatchErr = errors.New(`member must satisfy regular expression pattern: (brn:(cloud[a-zA-Z-]*)?:faas:)?([a-z]{2,5}[0-9]*:)?([0-9a-z]{32}:)?(function:)?([a-zA-Z0-9-_]+)(:(\$LATEST|[a-zA-Z0-9-_]+))?`)
)

type FunctionBRN struct {
	BRN
	FunctionName string
	Version      string
	Alias        string
}

var invalidFunctionBrnError = errors.New("invalid function brn")

func GenerateFunctionBRN(region, uid, functionName, qualifier string) FunctionBRN {
	var resource string
	var alias string
	var version string
	if len(qualifier) > 0 {
		resource = fmt.Sprintf("function:%s:%s", functionName, qualifier)
		version, alias = GetVersionAndAlias(qualifier)
	} else {
		resource = fmt.Sprintf("function:%s", functionName)
	}

	b := FunctionBRN{
		BRN: BRN{
			Partition: "cloud",
			Service:   "faas",
			Region:    region,
			AccountID: Md5BceUid(uid),
			Resource:  resource,
		},
		FunctionName: functionName,
		Version:      version,
		Alias:        alias,
	}

	return b
}

func ParseFunction(brn string) (FunctionBRN, error) {
	commonBrn, err := Parse(brn)
	if err != nil {
		return FunctionBRN{}, err
	}

	parts := strings.SplitN(commonBrn.Resource, ":", 3)
	if len(parts) < 2 {
		return FunctionBRN{}, invalidFunctionBrnError
	}
	if parts[0] != "function" {
		return FunctionBRN{}, invalidFunctionBrnError
	}
	functionName := parts[1]
	var qualifier string
	var alias string
	var version string
	if len(parts) == 3 {
		qualifier = parts[2]
		version, alias = GetVersionAndAlias(qualifier)
	}

	b := FunctionBRN{
		BRN:          commonBrn,
		FunctionName: functionName,
		Version:      version,
		Alias:        alias,
	}

	return b, nil
}

// fName = ThumbnailName/Uid-ThumbnailName/FuncBrn
func DealFName(accountID, fName string) (thumbnailName, version, alias string, err error) {

	t, err := JudgeUnderlyingBrn(fName)
	if err != nil {
		return
	}

	switch t {
	case "FunctionBRN":
		brn, errp := ParseFunction(fName)
		if errp != nil {
			err = errp
			return
		}
		thumbnailName = brn.FunctionName
		version = brn.Version
		alias = brn.Alias
		return
	case "PartialBRN":
		splitArr := strings.Split(fName, ":")
		lenSplitArr := len(splitArr)
		switch lenSplitArr {
		case 1:
			// function-name
			thumbnailName = splitArr[0]
		case 2:
			// account-id:function or function:version
			if accountID != "" && strings.HasPrefix(fName, accountID) {
				thumbnailName = splitArr[1]
			} else {
				thumbnailName = splitArr[0]
				version, alias = GetVersionAndAlias(splitArr[1])
			}
		case 3:
			// account-id:function:version
			if accountID != "" && accountID != splitArr[0] {
				err = RegNotMatchErr
			} else {
				accountID = splitArr[0]
				thumbnailName = splitArr[1]
				version, alias = GetVersionAndAlias(splitArr[2])
			}
		default:
			err = RegNotMatchErr
		}
		return
	}
	err = RegNotMatchErr
	return
}

func JudgeUnderlyingBrn(underlyingBrn string) (string, error) {
	if regBrn.MatchString(underlyingBrn) {
		return "FunctionBRN", nil
	} else if regPartialBrn.MatchString(underlyingBrn) {
		return "PartialBRN", nil
	} else {
		return "", RegNotMatchErr
	}
}

func JudgeQualifier(qualifier string) (string, error) {
	if api.RegVersion.MatchString(qualifier) {
		return "version", nil
	} else if RegAlias.MatchString(qualifier) {
		return "alias", nil
	} else {
		return "", errors.New("Qualifier not match Regexp")
	}
}

func GetVersionAndAlias(qualifier string) (version, alias string) {
	qtype, err := JudgeQualifier(qualifier)
	if err != nil {
		return
	} else if qtype == "version" {
		version = qualifier
		return
	} else if qtype == "alias" {
		alias = qualifier
		return
	}
	return
}

func GenerateFuncBrnString(region, uid, functionName, qualifier string) string {
	return GenerateFunctionBRN(region, uid, functionName, qualifier).String()
}
