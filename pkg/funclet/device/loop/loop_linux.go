//+build linux
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

// Package device
package loop

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/baidu/easyfaas/pkg/funclet/device/utils"

	"golang.org/x/sys/unix"
)

// open returns a file handle to /dev/loop# and returns an error if it cannot
// be opened.
func (device Device) open() (*os.File, error) {
	return os.OpenFile(device.Path(), device.flags, 0660)
}

// Path returns the path to the loopback device
func (device Device) Path() string {
	return fmt.Sprintf(DeviceFormatString, device.number)
}

func NewDevice(number uint64, flags int) Device {
	return Device{
		number: number,
		flags:  flags,
	}
}

// String implements the Stringer interface for Device
func (device Device) String() string {
	return device.Path()
}

// GetFree searches for the first free loopback device. If it cannot find one,
// it will attempt to create one. If anything fails, GetFree will return an
// error.
func GetFree() (Device, error) {
	ctrl, err := os.OpenFile(LoopControlPath, os.O_RDWR, 0660)
	if err != nil {
		return Device{}, fmt.Errorf("could not open %v: %v", LoopControlPath, err)
	}
	defer ctrl.Close()
	dev, _, errno := unix.Syscall(unix.SYS_IOCTL, ctrl.Fd(), CtlGetFree, 0)
	if dev < 0 {
		return Device{}, fmt.Errorf("could not get free device (err: %d): %v", errno, errno)
	}
	return Device{number: uint64(dev), flags: os.O_RDWR}, nil
}

// Attach attaches backingFile to the loopback device starting at offset. If ro
// is true, then the file is attached read only.
func Attach(backingFile string, offset uint64, ro bool) (Device, error) {
	var dev Device

	flags := os.O_RDWR
	if ro {
		flags = os.O_RDONLY
	}

	back, err := os.OpenFile(backingFile, flags, 0660)
	if err != nil {
		return dev, fmt.Errorf("could not open backing file: %v", err)
	}
	defer back.Close()

	dev, err = GetFree()
	if err != nil {
		return dev, err
	}
	dev.flags = flags

	if _, err := os.Stat(dev.Path()); err != nil {
		if err := syscall.Mknod(dev.Path(), syscall.S_IFBLK|0660, int(unix.Mkdev(uint32(Major), uint32(dev.number)))); err != nil {
			return dev, err
		}
	}
	loopFile, err := dev.open()
	if err != nil {
		return dev, fmt.Errorf("could not open loop device: %v", err)
	}
	defer loopFile.Close()

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, loopFile.Fd(), SetFd, back.Fd())
	if errno == 0 {
		info := Info{}
		copy(info.FileName[:], []byte(backingFile))
		info.Offset = offset
		if err := setInfo(loopFile.Fd(), info); err != nil {
			unix.Syscall(unix.SYS_IOCTL, loopFile.Fd(), ClrFd, 0)
			return dev, fmt.Errorf("could not set info")
		}
	}
	return dev, nil
}

// Detach removes the file backing the device.
func (device Device) Detach() error {
	return detach(device.Path())
}

func detach(fileName string) error {
	loopFile, err := os.OpenFile(fileName, os.O_RDONLY, 0660)
	if err != nil {
		return fmt.Errorf("could not open loop device")
	}
	defer loopFile.Close()

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, loopFile.Fd(), ClrFd, 0)
	if errno != 0 {
		return fmt.Errorf("error clearing loopfile: %v", errno)
	}

	return nil
}

func setInfo(fd uintptr, info Info) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, SetStatus64, uintptr(unsafe.Pointer(&info)))
	if errno == unix.ENXIO {
		return fmt.Errorf("device not backed by a file")
	} else if errno != 0 {
		return fmt.Errorf("could not get info about %v (err: %d): %v", fd, errno, errno)
	}

	return nil
}

func GetLoopDeviceMap() (map[string][]*Device, error) {
	deviceMap := make(map[string][]*Device, 0)
	cmd := exec.Command("losetup", "-a")
	cmd.Env = []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
	o, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	s := strings.Split(string(o), "\n")
	for _, v := range s {
		v = strings.TrimSpace(v)
		if len(v) == 0 {
			continue
		}
		if !strings.HasPrefix(v, "/dev/loop") {
			continue
		}
		loopInfoSlice := strings.Split(strings.TrimSpace(v), " ")
		if len(loopInfoSlice) < 3 {
			continue
		}
		numStr := strings.Trim(strings.TrimLeft(loopInfoSlice[0], "/dev/loop"), ":")
		if numStr == "" {
			continue
		}
		loopNum, err := strconv.ParseUint(numStr, 10, 64)
		if err != nil {
			continue
		}
		device := NewDevice(loopNum, os.O_RDWR)
		deviceID, err := utils.GetDevice(device.String())
		if err != nil {
			continue
		}
		device.Rdev = deviceID
		fPath := strings.TrimRight(strings.TrimLeft(loopInfoSlice[2], "("), ")")

		if len(loopInfoSlice) == 4 && strings.ContainsAny(loopInfoSlice[3], "(deleted)") {
			device.IsDeleted = true
		}

		if deviceList, ok := deviceMap[loopInfoSlice[2]]; !ok {
			deviceMap[fPath] = []*Device{&device}
		} else {
			deviceMap[fPath] = append(deviceList, &device)
		}
	}
	return deviceMap, nil
}
