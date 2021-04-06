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

# The root of the build/dist directory
KUN_ROOT="$(cd "$(dirname "${BASH_SOURCE}")/../.." && pwd -P)"

KUN_OUTPUT_SUBPATH="${KUN_OUTPUT_SUBPATH:-_output/local}"
KUN_OUTPUT="${KUN_ROOT}/${KUN_OUTPUT_SUBPATH}"
KUN_OUTPUT_BINPATH="${KUN_OUTPUT}/bin"

# This controls rsync compression. Set to a value > 0 to enable rsync
# compression for build container
KUN_RSYNC_COMPRESS="${KUN_RSYNC_COMPRESS:-0}"

# Set no_proxy for localhost if behind a proxy, otherwise,
# the connections to localhost in scripts will time out
export no_proxy=127.0.0.1,localhost

# This is a symlink to binaries for "this platform", e.g. build tools.
THIS_PLATFORM_BIN="${KUN_ROOT}/_output/bin"

source "${KUN_ROOT}/build/lib/util.sh"
source "${KUN_ROOT}/build/lib/logging.sh"

kun::log::install_errexit

source "${KUN_ROOT}/build/lib/version.sh"
source "${KUN_ROOT}/build/lib/golang.sh"

KUN_OUTPUT_HOSTBIN="${KUN_OUTPUT_BINPATH}/$(kun::util::host_platform)"

# list of all available group versions.  This should be used when generated code
# or when starting an API server that you want to have everything.
# most preferred version for a group should appear first
KUN_AVAILABLE_GROUP_VERSIONS="${KUN_AVAILABLE_GROUP_VERSIONS:-\
v1 \
}"
# not all group versions are exposed by the server.  This list contains those
# which are not available so we don't generate clients or swagger for them
KUN_NONSERVER_GROUP_VERSIONS="
 abac.authorization.kunrnetes.io/v0 \
 abac.authorization.kunrnetes.io/v1beta1 \
 componentconfig/v1alpha1 \
 imagepolicy.k8s.io/v1alpha1\
 admission.k8s.io/v1alpha1\
"

# This emulates "readlink -f" which is not available on MacOS X.
# Test:
# T=/tmp/$$.$RANDOM
# mkdir $T
# touch $T/file
# mkdir $T/dir
# ln -s $T/file $T/linkfile
# ln -s $T/dir $T/linkdir
# function testone() {
#   X=$(readlink -f $1 2>&1)
#   Y=$(kun::readlinkdashf $1 2>&1)
#   if [ "$X" != "$Y" ]; then
#     echo readlinkdashf $1: expected "$X", got "$Y"
#   fi
# }
# testone /
# testone /tmp
# testone $T
# testone $T/file
# testone $T/dir
# testone $T/linkfile
# testone $T/linkdir
# testone $T/nonexistant
# testone $T/linkdir/file
# testone $T/linkdir/dir
# testone $T/linkdir/linkfile
# testone $T/linkdir/linkdir
function kun::readlinkdashf {
  # run in a subshell for simpler 'cd'
  (
    if [[ -d "$1" ]]; then # This also catch symlinks to dirs.
      cd "$1"
      pwd -P
    else
      cd $(dirname "$1")
      local f
      f=$(basename "$1")
      if [[ -L "$f" ]]; then
        readlink "$f"
      else
        echo "$(pwd -P)/${f}"
      fi
    fi
  )
}

# This emulates "realpath" which is not available on MacOS X
# Test:
# T=/tmp/$$.$RANDOM
# mkdir $T
# touch $T/file
# mkdir $T/dir
# ln -s $T/file $T/linkfile
# ln -s $T/dir $T/linkdir
# function testone() {
#   X=$(realpath $1 2>&1)
#   Y=$(kun::realpath $1 2>&1)
#   if [ "$X" != "$Y" ]; then
#     echo realpath $1: expected "$X", got "$Y"
#   fi
# }
# testone /
# testone /tmp
# testone $T
# testone $T/file
# testone $T/dir
# testone $T/linkfile
# testone $T/linkdir
# testone $T/nonexistant
# testone $T/linkdir/file
# testone $T/linkdir/dir
# testone $T/linkdir/linkfile
# testone $T/linkdir/linkdir
kun::realpath() {
  if [[ ! -e "$1" ]]; then
    echo "$1: No such file or directory" >&2
    return 1
  fi
  kun::readlinkdashf "$1"
}

