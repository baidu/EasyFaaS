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

set -e

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo "  -h  --help      帮助"
    echo "  -m= --module=   模块名称"
    echo "  -i= --image=    镜像名称"
    echo "  -t= --tag=      镜像标签"
    echo "  -e= --env=      部署环境（镜像仓库配置）"
    exit
}

MODULE=controller
IMAGE=controller-dev
WITHBUILD=${WITHBUILD:-0}

for i in "$@"
do
case $i in
    -h|--help)
    usage
    exit 0
    ;;
    -m=*|--module=*)
    MODULE=${i#*=}
    shift
    ;;
    -i=*|-image=*)
    IMAGE=${i#*=}
    shift
    ;;
    -t=*|--tag=*)
    TAG=${i#*=}
    shift
    ;;
    -e=*|--env=*)
    REGISTRY_ENV=${i#*=}
    shift
    ;;
    *)
    echo "unknown options"
    exit 1
    ;;
esac
done

if [ -n $REGISTRY ]; then
    CONFIG=""
elif [ "x$REGISTRY_ENV" = "xtest" ]; then
    if [ -z $DOCKER_CONFIG_TEST ]; then
        CONFIG="~/baidu/docker/config/test"
    else
        CONFIG=$DOCKER_CONFIG_TEST
    fi
    REGISTRY="registry.baidubce.com/openless-dev/"
else
    if [ -z $DOCKER_CONFIG_ONLINE ]; then
        CONFIG="~/baidu/docker/config/online"
    else
        CONFIG=$DOCKER_CONFIG_ONLINE
    fi
    REGISTRY="registry.baidubce.com/openless/"
fi

VCS_REF=$(git rev-parse --short HEAD)
if [ -z $TAG ]; then
  if git_status=$(git status --porcelain 2>/dev/null) && [[ -z ${git_status} ]]; then
    TAG=${VCS_REF}
  else
    TAG=${VCS_REF}-"dirty"
  fi
fi

IMAGE_NAME="$IMAGE:$TAG"
PRIVATE_IMAGE_NAME="$REGISTRY$IMAGE_NAME"

# get script base dir
pushd . > /dev/null
CWD="${BASH_SOURCE[0]}";
while([ -L "${CWD}" ]); do
    cd "`dirname "${CWD}"`"
    CWD="$(readlink "$(basename "${CWD}")")";
done
BUILD_HOME="$(dirname "${CWD}")"
BASEDIR="$(pwd)"
PROJ_NAME=""

if [[ "x$WITHBUILD" == "x1" ]]; then
    docker run -v $BASEDIR:/go/src/$PROJ_NAME  --rm registry.baidubce.com/openless-public/golang-build:v1.0 \
    bash -c "go env -w GO111MODULE='on' \\
    && go env -w GONOPROXY=\*\*.baidu.com\*\*  \\
    && go env -w GONOSUMDB=\* \\
    && go env -w GOPROXY=https://goproxy.baidu.com \\
    && go env -w GOPRIVATE=\*.baidu.com \\
    && cd /go/src/$PROJ_NAME && make WHAT=cmd/$MODULE"
fi

if [[ "$MODULE" == "one" ]]; then
  cp "$BUILD_HOME/../_output/local/bin/linux/amd64/"* "$BUILD_HOME/allInOne/docker/"
  docker build -t $IMAGE_NAME "$BUILD_HOME/allInOne/docker"
else
  cp "$BUILD_HOME/../_output/local/bin/linux/amd64/$MODULE" "$BUILD_HOME/$MODULE/docker/"
  docker build -t $IMAGE_NAME "$BUILD_HOME/$MODULE/docker"
fi
docker tag $IMAGE_NAME $PRIVATE_IMAGE_NAME

[ $? != 0 ] && \
  error "Docker image build failed !" && exit 100
if [ "$CONFIG" == "" ]; then
  echo "Use docker push $PRIVATE_IMAGE_NAME to publish image"
else
  echo "Use docker --config $CONFIG push $PRIVATE_IMAGE_NAME to publish image"
fi