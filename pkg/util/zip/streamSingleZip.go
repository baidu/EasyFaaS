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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"
)

type mockedFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	os.FileInfo
}

func (f *mockedFileInfo) Name() string {
	return f.name
}

func (f *mockedFileInfo) Size() int64 {
	return f.size
}

func (f *mockedFileInfo) Mode() os.FileMode {
	return f.mode
}

func (f *mockedFileInfo) ModTime() time.Time {
	return f.modTime
}

// StreamSingleUnzip
func StreamSingleUnzip(b []byte, fileName string) ([]byte, error) {
	r := bytes.NewReader(b)
	zipReader, err := zip.NewReader(r, int64(len(b)))
	if err != nil {
		return nil, err
	}
	for _, file := range zipReader.File {
		if file.FileHeader.Name == fileName {
			f, err := file.Open()
			defer f.Close()
			if err != nil {
				return nil, err
			}
			data, err := ioutil.ReadAll(f)
			if err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	errStr := fmt.Sprintf("ERROR Happend: %s was not found in zip, please check you zip file and your functions's handle config", fileName)
	// return nil, fmt.Errorf("%s was not found in zip", fileName)
	return []byte(errStr), nil
}

// StreamSingleZip
func StreamSingleZip(sourceName string, data []byte) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	zipWriter := zip.NewWriter(buffer)

	info := &mockedFileInfo{
		name:    sourceName,
		size:    int64(len(data)),
		mode:    0777,
		modTime: time.Now(),
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		zipWriter.Close()
		return nil, err
	}

	// Change to deflate to gain better compression
	// see http://golang.org/pkg/archive/zip/#pkg-constants
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		zipWriter.Close()
		return nil, err
	}
	_, err = writer.Write(data)

	if err != nil {
		zipWriter.Close()
		return nil, err
	}
	zipWriter.Close()
	return buffer.Bytes(), nil
}

func UpdateZipCode(b []byte, files []*map[string]string) ([]byte, error) {
	bufferc := bytes.NewBuffer(nil)
	zipWriter := zip.NewWriter(bufferc)

	var err error
	defer func() {
		if err != nil {
			zipWriter.Close()
		}
	}()

	//read
	r := bytes.NewReader(b)
	zipReader, err := zip.NewReader(r, int64(len(b)))
	if err != nil {
		return nil, err
	}
	for _, file := range zipReader.File {
		needCopy := true
		for _, wfile := range files {
			if file.FileHeader.Name == (*wfile)["fileName"] {
				needCopy = false
			}
		}
		if needCopy == true {
			f, err := file.Open()
			if err != nil {
				return nil, err
			}
			writer, err := zipWriter.CreateHeader(&file.FileHeader)
			if err != nil {
				return nil, err
			}
			_, err = io.Copy(writer, f)

			if err != nil {
				return nil, err
			}
			f.Close()
		}

	}
	for _, wfile := range files {
		writer, err := zipWriter.Create((*wfile)["fileName"])
		if err != nil {
			return nil, err
		}
		_, err = writer.Write([]byte((*wfile)["fileBytes"]))
		if err != nil {
			return nil, err
		}
	}

	zipWriter.Close()
	return bufferc.Bytes(), nil
}
