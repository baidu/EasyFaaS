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

package code

import (
	"os"
	"path/filepath"
)

// FindCodeCompeleteTag
func (codeMgr *Manager) FindCodeCompeleteTag(path, codeSha256 string) bool {
	tagName := getCompleteTag(path, codeSha256)
	if _, err := os.Stat(tagName); os.IsNotExist(err) {
		return false
	}
	return true
}

// CreateCodeCompeleteTag
func (codeMgr *Manager) CreateCodeCompeleteTag(path, codeSha256 string) error {
	tagName := getCompleteTag(path, codeSha256)
	_, err := os.Create(tagName)
	if err != nil {
		return err
	}
	return nil
}

// RemoveCodeCompeleteTag
func (codeMgr *Manager) RemoveCodeCompeleteTag(path, codeSha256 string) error {
	tagName := getCompleteTag(path, codeSha256)
	err := os.Remove(tagName)
	if err != nil {
		return err
	}
	return nil
}

func getCompleteTag(path, codeSha256 string) string {
	return filepath.Join(path, codeSha256+".COMPLETE")
}
