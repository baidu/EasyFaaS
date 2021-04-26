//+build !linux

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



// Path returns the path to the loopback device
func (device Device) Path() string {
	return ""
}

func NewDevice(number uint64, flags int) Device {
	return Device{
		number: number,
		flags:  flags,
	}
}

// String implements the Stringer interface for Device
func (device Device) String() string {
	return ""
}

// GetFree searches for the first free loopback device. If it cannot find one,
// it will attempt to create one. If anything fails, GetFree will return an
// error.
func GetFree() (Device, error) {
	return Device{}, nil
}

// Attach attaches backingFile to the loopback device starting at offset. If ro
// is true, then the file is attached read only.
func Attach(backingFile string, offset uint64, ro bool) (Device, error) {
	return Device{}, nil
}

// Detach removes the file backing the device.
func (device Device) Detach() error {
	return detach(device.Path())
}

func detach(fileName string) error {
	return nil
}

func setInfo(fd uintptr, info Info) error {
	return nil
}

func GetLoopDeviceMap() (map[string][]*Device, error) {
	return nil, nil
}
