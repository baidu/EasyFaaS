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

package mount

// Info reveals information about a particular mounted filesystem. This
// struct is populated from the content in the /proc/<pid>/mountinfo file.
type Info struct {
	// ID is a unique identifier of the mount (may be reused after umount).
	ID int

	// Parent indicates the ID of the mount parent (or of self for the top of the
	// mount tree).
	Parent int

	// Major indicates one half of the device ID which identifies the device class.
	Major int

	// Minor indicates one half of the device ID which identifies a specific
	// instance of device.
	Minor int

	// Root of the mount within the filesystem.
	Root string

	// Mountpoint indicates the mount point relative to the process's root.
	Mountpoint string

	// Opts represents mount-specific options.
	Opts string

	// Optional represents optional fields.
	Optional string

	// Fstype indicates the type of filesystem, such as EXT3.
	Fstype string

	// Source indicates filesystem specific information or "none".
	Source string

	// VfsOpts represents per super block options.
	VfsOpts string
}

type byMountpoint []*Info

func (by byMountpoint) Len() int {
	return len(by)
}

func (by byMountpoint) Less(i, j int) bool {
	return by[i].Mountpoint < by[j].Mountpoint
}

func (by byMountpoint) Swap(i, j int) {
	by[i], by[j] = by[j], by[i]
}
