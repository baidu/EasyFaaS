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

package zip

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	UnzipError = errors.New("unzip cross border")
)

func Unzip(archive, target string) error {
	if !path.IsAbs(archive) || !path.IsAbs(target) {
		return errors.New("archive and target path is not absolute")
	}
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	err = unzip(reader, target)
	if err != nil {
		return err
	}
	return nil
}

func unzip(reader *zip.ReadCloser, target string) error {
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		fPath := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			mkdirAll(fPath, target)
			continue
		} else {
			if fDir := path.Dir(fPath); fDir != target {
				if err := mkdirAll(fDir, target); err != nil {
					if err == UnzipError {
						continue
					}
					return err
				}
			}
		}

		err := ioCopy(file, fPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func mkdirAll(fDir, target string) error {
	if !strings.HasPrefix(fDir, target) {
		return UnzipError
	}
	return os.MkdirAll(fDir, 0755)
}

func ioCopy(file *zip.File, fPath string) error {

	fileReader, err := file.Open()
	if err != nil {
		return err
	}
	defer fileReader.Close()

	targetFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, fileReader); err != nil {
		return err
	}
	return nil
}

func UnzipFromBytes(targetDir string, b []byte, length int64) error {
	zr, err := zip.NewReader(bytes.NewReader(b), length)
	if err != nil {
		return err
	}
	rc := &zip.ReadCloser{
		Reader: *zr,
	}
	return unzip(rc, targetDir)
}

func GetUnzipFileNum(b []byte, length int64) (int, error) {
	zr, err := zip.NewReader(bytes.NewReader(b), length)
	if err != nil {
		return 0, err
	}
	return len(zr.File), nil
}
