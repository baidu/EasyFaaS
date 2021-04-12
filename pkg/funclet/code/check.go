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
	"crypto/sha256"
	"encoding/base64"
	"io"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/baidu/easyfaas/pkg/funclet/context"
)

// CheckCode
func (codeMgr *Manager) CheckCode(ctx *context.Context, filename, codeSha256 string) bool {
	defer ctx.Logger.TimeTrack(time.Now(), "Checksum", zap.String("filename", filename))
	s, err := codeMgr.calSHA256(ctx, filename)
	if err != nil {
		return false
	}
	if s != codeSha256 {
		ctx.Logger.Warnf("Check code failed, expected %s, got %s", codeSha256, s)
		return false
	}
	return true
}

func (codeMgr *Manager) calSHA256(ctx *context.Context, filename string) (string, error) {
	hasher := sha256.New()
	f, err := os.Open(filename)
	if err != nil {
		ctx.Logger.Warnf("Open file failed: %v", err)
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		ctx.Logger.Warnf("Read file failed: %v", err)
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil)), nil
}
