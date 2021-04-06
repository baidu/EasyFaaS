// +build !linux

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


package file

func (m *MountManager) MakeShared(path string) error {
	return nil
}

func (m *MountManager) BindMount(pairs []*MountPair, atomic bool) error {
	return nil
}

func (m *MountManager) Unmount(path []string, ignoreError bool) error {
	return nil
}

func (m *MountManager) UnmountAllByPathName(name string) error {
	return nil
}
