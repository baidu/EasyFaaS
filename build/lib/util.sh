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

kun::util::sortable_date() {
  date "+%Y%m%d-%H%M%S"
}

kun::util::wait_for_url() {
  local url=$1
  local prefix=${2:-}
  local wait=${3:-1}
  local times=${4:-30}

  which curl >/dev/null || {
    kun::log::usage "curl must be installed"
    exit 1
  }

  local i
  for i in $(seq 1 $times); do
    local out
    if out=$(curl --max-time 1 -gkfs $url 2>/dev/null); then
      kun::log::status "On try ${i}, ${prefix}: ${out}"
      return 0
    fi
    sleep ${wait}
  done
  kun::log::error "Timed out waiting for ${prefix} to answer at ${url}; tried ${times} waiting ${wait} between each"
  return 1
}

# returns a random port
kun::util::get_random_port() {
  awk -v min=1024 -v max=65535 'BEGIN{srand(); print int(min+rand()*(max-min+1))}'
}

# use netcat to check if the host($1):port($2) is free (return 0 means free, 1 means used)
kun::util::test_host_port_free() {
  local host=$1
  local port=$2
  local success=0
  local fail=1

  which nc >/dev/null || {
    kun::log::usage "netcat isn't installed, can't verify if ${host}:${port} is free, skipping the check..."
    return ${success}
  }

  if [ ! $(nc -vz "${host}" "${port}") ]; then
    kun::log::status "${host}:${port} is free, proceeding..."
    return ${success}
  else
    kun::log::status "${host}:${port} is already used"
    return ${fail}
  fi
}

# Example:  kun::util::trap_add 'echo "in trap DEBUG"' DEBUG
# See: http://stackoverflow.com/questions/3338030/multiple-bash-traps-for-the-same-signal
kun::util::trap_add() {
  local trap_add_cmd
  trap_add_cmd=$1
  shift

  for trap_add_name in "$@"; do
    local existing_cmd
    local new_cmd

    # Grab the currently defined trap commands for this trap
    existing_cmd=`trap -p "${trap_add_name}" |  awk -F"'" '{print $2}'`

    if [[ -z "${existing_cmd}" ]]; then
      new_cmd="${trap_add_cmd}"
    else
      new_cmd="${existing_cmd};${trap_add_cmd}"
    fi

    # Assign the test
    trap "${new_cmd}" "${trap_add_name}"
  done
}

# Opposite of kun::util::ensure-temp-dir()
kun::util::cleanup-temp-dir() {
  rm -rf "${KUN_TEMP}"
}

# Create a temp dir that'll be deleted at the end of this bash session.
#
# Vars set:
#   KUN_TEMP
kun::util::ensure-temp-dir() {
  if [[ -z ${KUN_TEMP-} ]]; then
    KUN_TEMP=$(mktemp -d 2>/dev/null || mktemp -d -t kun.XXXXXX)
    kun::util::trap_add kun::util::cleanup-temp-dir EXIT
  fi
}

# This figures out the host platform without relying on golang.  We need this as
# we don't want a golang install to be a prerequisite to building yet we need
# this info to figure out where the final binaries are placed.
kun::util::host_platform() {
  local host_os
  local host_arch
  case "$(uname -s)" in
    Darwin)
      host_os=darwin
      ;;
    Linux)
      host_os=linux
      ;;
    *)
      kun::log::error "Unsupported host OS.  Must be Linux or Mac OS X."
      exit 1
      ;;
  esac

  case "$(uname -m)" in
    x86_64*)
      host_arch=amd64
      ;;
    i?86_64*)
      host_arch=amd64
      ;;
    amd64*)
      host_arch=amd64
      ;;
    aarch64*)
      host_arch=arm64
      ;;
    arm64*)
      host_arch=arm64
      ;;
    arm*)
      host_arch=arm
      ;;
    i?86*)
      host_arch=x86
      ;;
    s390x*)
      host_arch=s390x
      ;;
    ppc64le*)
      host_arch=ppc64le
      ;;
    *)
      kun::log::error "Unsupported host arch. Must be x86_64, 386, arm, arm64, s390x or ppc64le."
      exit 1
      ;;
  esac
  echo "${host_os}/${host_arch}"
}

kun::util::find-binary-for-platform() {
  local -r lookfor="$1"
  local -r platform="$2"
  local locations=(
    "${KUN_ROOT}/_output/bin/${lookfor}"
    "${KUN_ROOT}/_output/dockerized/bin/${platform}/${lookfor}"
    "${KUN_ROOT}/_output/local/bin/${platform}/${lookfor}"
    "${KUN_ROOT}/platforms/${platform}/${lookfor}"
  )
  # Also search for binary in bazel build tree.
  # In some cases we have to name the binary $BINARY_bin, since there was a
  # directory named $BINARY next to it.
  locations+=($(find "${KUN_ROOT}/bazel-bin/" -type f -executable \
    \( -name "${lookfor}" -o -name "${lookfor}_bin" \) 2>/dev/null || true) )

  # List most recently-updated location.
  local -r bin=$( (ls -t "${locations[@]}" 2>/dev/null || true) | head -1 )
  echo -n "${bin}"
}

kun::util::find-binary() {
  kun::util::find-binary-for-platform "$1" "$(kun::util::host_platform)"
}

# Some useful colors.
if [[ -z "${color_start-}" ]]; then
  declare -r color_start="\033["
  declare -r color_red="${color_start}0;31m"
  declare -r color_yellow="${color_start}0;33m"
  declare -r color_green="${color_start}0;32m"
  declare -r color_norm="${color_start}0m"
fi

# ex: ts=2 sw=2 et filetype=sh
