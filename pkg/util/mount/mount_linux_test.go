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

import (
	"os"
	"path"
	"testing"
)

func TestMounted(t *testing.T) {
	tmp := path.Join(os.TempDir(), "mount-tests")
	if err := os.MkdirAll(tmp, 0777); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	var (
		sourceDir  = path.Join(tmp, "source")
		targetDir  = path.Join(tmp, "target")
		sourcePath = path.Join(sourceDir, "file.txt")
		targetPath = path.Join(targetDir, "file.txt")
	)

	os.Mkdir(sourceDir, 0777)
	os.Mkdir(targetDir, 0777)

	f, err := os.Create(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("hello")
	f.Close()

	f, err = os.Create(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	if err := BindMount(sourceDir, targetDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := Unmount(targetDir); err != nil {
			t.Fatal(err)
		}
	}()

	mounted, err := Mounted(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	if !mounted {
		t.Fatalf("Expected %s to be mounted", targetDir)
	}
	if _, err := os.Stat(targetDir); err != nil {
		t.Fatal(err)
	}
}
