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

package funclet

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/baidu/easyfaas/pkg/funclet/tmp"

	"github.com/baidu/easyfaas/cmd/funclet/options"
	"github.com/baidu/easyfaas/pkg/funclet/file"
	"github.com/baidu/easyfaas/pkg/util/logs"
)

func GetPathConfig(o *options.FuncletOptions) *file.PathConfig {
	return &file.PathConfig{
		RunnerDataPath:    o.RunnerDataPath,
		RunnerSpecPath:    o.RunnerSpecPath,
		RunnerTmpPath:     tmp.RunnersTmpPath,
		CodeWorkspacePath: tmp.CodeWorkspacePath,
		EtcPath:           o.RunnerSpecOption.EtcPath,
		ConfPath:          o.RunnerSpecOption.ConfPath,
		CodePath:          o.RunnerSpecOption.CodePath,
		RuntimePath:       o.RunnerSpecOption.RuntimePath,
		RunningMode:       o.RunningMode,
	}
}

func ReaderToLog(r io.ReadCloser, logger *logs.Logger) {
	logger.Info("start logger")
	reader := bufio.NewReader(r)
	defer func() {
		logger.V(4).Info("stop logger")
		r.Close()
	}()
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				logger.V(4).Infof("read from pipe %s", err)
				break
			} else {
				logger.Errorf("read from pipe failed: %s", err)
			}
		}
		logger.V(4).Info(string(line))
	}
	return
}

func checkRootfs(timeout time.Duration, rootfsPath string, logger *logs.Logger) error {
	t := time.NewTicker(timeout)
Loop:
	for {
		select {
		case <-t.C:
			logger.Errorf("waiting for rootfs timeout for 1 min")
			return errors.New("prepare rootfs error ")
		default:
			if _, err := os.Stat(rootfsPath); err != nil {
				if os.IsNotExist(err) {
					logger.Infof("waiting for rootfs...")
					time.Sleep(time.Second)
				} else {
					return errors.New("prepare rootfs error ")
				}
			} else {
				break Loop
			}
		}
	}
	return nil
}

func generateContainerID(podName string, num int) string {
	// TODO: change to short uuid
	// this containerID is runtime's hostname
	// so it must be shorter than 64 bytes
	return podName +
		"-controller" +
		"-c" + strconv.Itoa(num)
}

func CopyFile(src, dst string, mode os.FileMode) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}

	if mode != 0 {
		err = os.Chmod(dst, mode)
	} else {
		err = os.Chmod(dst, si.Mode())
	}
	if err != nil {
		return
	}

	return
}

func CopyDir(src string, dst string, mode os.FileMode) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if err == nil {
		return fmt.Errorf("destination already exists")
	}
	if mode != 0 {
		err = os.MkdirAll(dst, mode)
	} else {
		err = os.MkdirAll(dst, si.Mode())
	}
	if err != nil {
		return
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath, mode)
			if err != nil {
				logs.Errorf("copy dir failed: %s", err)
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(srcPath, dstPath, mode)
			if err != nil {
				logs.Errorf("copy file failed: %s", err)
				return
			}
		}
	}

	return
}
