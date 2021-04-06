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
	"strings"
)

// SimplifyFunctionNameAndQualifier
// TODO: partial function name
func SimplifyFunctionNameAndQualifier(functionName, qualifier string) (string, string) {
	functionBrn, err := Parse(functionName)
	if err != nil {
		return functionName, qualifier
	}
	functionName = functionBrn.Resource
	if strings.Contains(functionName, ":") {
		sep := strings.Split(functionName, ":")
		if len(sep) == 2 {
			functionName = sep[1]
			if len(qualifier) == 0 {
				qualifier = "LATEST"
			}
		} else if len(sep) == 3 {
			functionName = sep[1]
			qualifier = sep[2]
		}
	}
	if qualifier == "$LATEST" {
		qualifier = "LATEST"
	}
	return functionName, qualifier
}

//fName = ThumbnailName/Uid-ThumbnailName/FuncBrn
func DealFName1(accountID, fName string) (thumbnailName, version, uidStr string, err error) {
	if accountID != "" {
		uidStr = accountID
	}
	if regBrn.MatchString(fName) {
		var brn BRN
		brn, err = Parse(fName)
		if err != nil {
			return
		}
		uidStr = brn.AccountID
		splitRes := strings.Split(brn.Resource, ":")
		if len(splitRes) == 2 {
			thumbnailName = splitRes[1]
		} else if len(splitRes) == 3 {
			thumbnailName = splitRes[1]
			version = splitRes[2]
		} else {
			err = RegNotMatchErr
		}
		return
	} else if regPartialBrn.MatchString(fName) {
		splitArr := strings.Split(fName, ":")
		lenSplitArr := len(splitArr)
		switch lenSplitArr {
		case 1:
			//function-name
			thumbnailName = splitArr[0]
		case 2:
			// account-id:function or function:version
			if accountID != "" && strings.HasPrefix(fName, accountID) {
				thumbnailName = splitArr[1]
			} else {
				thumbnailName = splitArr[0]
				version = splitArr[1]
			}
		case 3:
			// account-id:function:version
			if accountID != "" && accountID != splitArr[0] {
				err = RegNotMatchErr
			} else {
				accountID = splitArr[0]
				thumbnailName = splitArr[1]
				version = splitArr[2]
			}
		default:
			err = RegNotMatchErr
		}
		return
	}
	err = RegNotMatchErr
	return
}
