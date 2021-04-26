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

package loop


const (
	// general constants
	NameSize = 64
	KeySize  = 32
	Major    = 7

	// paths
	LoopControlPath = "/dev/loop-control"

	// DeviceFormatString holds the format of loopback devices
	DeviceFormatString = "/dev/loop%d"

	// ioctl commands
	SetFd = 0x4C00
	ClrFd = 0x4C01

	SetStatus64 = 0x4C04

	CtlAdd     = 0x4C80
	CtlRemove  = 0x4C81
	CtlGetFree = 0x4C82
)

// Info is a datastructure that holds relevant information about a file backed
// loopback device.
type Info struct {
	Device         uint64
	INode          uint64
	RDevice        uint64
	Offset         uint64
	SizeLimit      uint64
	Number         uint32
	EncryptType    uint32
	EncryptKeySize uint32
	Flags          uint32
	FileName       [NameSize]byte
	CryptName      [NameSize]byte
	EncryptKey     [KeySize]byte
	Init           [2]uint64
}

// Device represents a loop device /dev/loop#
type Device struct {
	// device number (i.e. 7 if /dev/loop7)
	number uint64

	// flags with which to open the device with
	flags int

	Rdev uint64

	IsDeleted bool
}
