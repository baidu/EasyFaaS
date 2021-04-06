#!/bin/bash

# Copyright (c) 2020 Baidu, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

# download go 1.13.5
echo "Download go for building..."
mkdir build/golang-dl
pushd build
pushd golang-dl
curl -Lk -H 'IREPO-TOKEN:09f6f1f8-ab6d-42ba-a393-326e0b955ae4' "https://irepo.baidu-int.com/rest/prod/v3/baidu/bce-bap/golang-dl/releases/1.13.5.1/files" | tar xzv
mv output/go1.13.5.linux-amd64.tar.gz ./
rm -rf output/
tar xzf go1.13.5.linux-amd64.tar.gz
export GOROOT=$(pwd)/go
export PATH=$GOROOT/bin:$PATH
$GOROOT/bin/go env -w GONOPROXY="**.baidu.com**"
$GOROOT/bin/go env -w GONOSUMDB="**.baidu.com**"
$GOROOT/bin/go env -w GOPROXY="https://goproxy.bj.bcebos.com,direct"
popd
mkdir gopath
export GOPATH=$(pwd)/gopath
go env
# ensure icode package
git config --global url.ssh://git@icode.baidu.com:8235/baidu/.insteadof https://icode.baidu.com/baidu/
echo "Go downloaded, start making"

# build controller
popd
make clean
make

rm -rf output
mkdir -p output
cp _output/local/bin/linux/amd64/* output

echo "Build end"
