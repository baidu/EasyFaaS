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

readonly ORIGIN_GOPATH="${GOPATH:-}"

# The golang package that we are building.
readonly KUN_GO_PACKAGE=github.com/baidu/easyfaas
readonly KUN_GOPATH="${KUN_OUTPUT}/go"

# The set of server targets that we are only building for Linux
# If you update this list, please also update build/BUILD.
kun::golang::server_targets() {
  local targets=(
    cmd/controller
    cmd/funclet
    cmd/stubs
    cmd/httptrigger
  )
  echo "${targets[@]}"
}

readonly KUN_SERVER_TARGETS=($(kun::golang::server_targets))
readonly KUN_SERVER_BINARIES=("${KUN_SERVER_TARGETS[@]##*/}")

if [[ -n "${KUN_BUILD_PLATFORMS:-}" ]]; then
  readonly KUN_SERVER_PLATFORMS=(${KUN_BUILD_PLATFORMS})
  readonly KUN_NODE_PLATFORMS=(${KUN_BUILD_PLATFORMS})
  readonly KUN_TEST_PLATFORMS=(${KUN_BUILD_PLATFORMS})
  readonly KUN_CLIENT_PLATFORMS=(${KUN_BUILD_PLATFORMS})
elif [[ "${KUN_FASTBUILD:-}" == "true" ]]; then
  readonly KUN_SERVER_PLATFORMS=(linux/amd64)
  readonly KUN_NODE_PLATFORMS=(linux/amd64)
  if [[ "${KUN_BUILDER_OS:-}" == "darwin"* ]]; then
    readonly KUN_TEST_PLATFORMS=(
      darwin/amd64
      linux/amd64
    )
    readonly KUN_CLIENT_PLATFORMS=(
      darwin/amd64
      linux/amd64
    )
  else
    readonly KUN_TEST_PLATFORMS=(linux/amd64)
    readonly KUN_CLIENT_PLATFORMS=(linux/amd64)
  fi
else

  # The server platform we are building on.
  readonly KUN_SERVER_PLATFORMS=(
    linux/amd64
    linux/arm
    linux/arm64
    linux/s390x
    linux/ppc64le
  )

  # The node platforms we build for
  readonly KUN_NODE_PLATFORMS=(
    linux/amd64
    linux/arm
    linux/arm64
    linux/s390x
    linux/ppc64le
    windows/amd64
  )

  # If we update this we should also update the set of platforms whose standard library is precompiled for in build/build-image/cross/Dockerfile
  readonly KUN_CLIENT_PLATFORMS=(
    linux/amd64
    linux/386
    linux/arm
    linux/arm64
    linux/s390x
    linux/ppc64le
    darwin/amd64
    darwin/386
    windows/amd64
    windows/386
  )

  # Which platforms we should compile test targets for. Not all client platforms need these tests
  readonly KUN_TEST_PLATFORMS=(
    linux/amd64
    darwin/amd64
    windows/amd64
  )
fi

# TODO(pipejakob) gke-certificates-controller is included here to exercise its
# compilation, but it doesn't need to be distributed in any of our tars. Its
# code is only living in this repo temporarily until it finds a new home.
readonly KUN_ALL_TARGETS=(
  "${KUN_SERVER_TARGETS[@]}"
)

kun::golang::is_statically_linked_library() {
  #local e
  #for e in "${KUN_STATIC_LIBRARIES[@]}"; do [[ "$1" == *"/$e" ]] && return 0; done;
  # Allow individual overrides--e.g., so that you can get a static build of
  # kunctl for inclusion in a container.
  #if [ -n "${KUN_STATIC_OVERRIDES:+x}" ]; then
  #for e in "${KUN_STATIC_OVERRIDES[@]}"; do [[ "$1" == *"/$e" ]] && return 0; done;
  #fi
  return 1
}

# kun::binaries_from_targets take a list of build targets and return the
# full go package to be built
kun::golang::binaries_from_targets() {
  local target
  for target; do
    # If the target starts with what looks like a domain name, assume it has a
    # fully-qualified package name rather than one that needs package prepended.
    if [[ "${target}" =~ ^([[:alnum:]]+".")+[[:alnum:]]+"/" ]]; then
      echo "${target}"
    else
      echo "${KUN_GO_PACKAGE}/${target}"
    fi
  done
}

# Asks golang what it thinks the host platform is. The go tool chain does some
# slightly different things when the target platform matches the host platform.
kun::golang::host_platform() {
  echo "$(go env GOHOSTOS)/$(go env GOHOSTARCH)"
}

kun::golang::current_platform() {
  local os="${GOOS-}"
  if [[ -z $os ]]; then
    os=$(go env GOHOSTOS)
  fi

  local arch="${GOARCH-}"
  if [[ -z $arch ]]; then
    arch=$(go env GOHOSTARCH)
  fi

  echo "$os/$arch"
}

# Takes the the platform name ($1) and sets the appropriate golang env variables
# for that platform.
kun::golang::set_platform_envs() {
  [[ -n ${1-} ]] || {
    kun::log::error_exit "!!! Internal error. No platform set in kun::golang::set_platform_envs"
  }

  export GOOS=${platform%/*}
  export GOARCH=${platform##*/}

  # Do not set CC when building natively on a platform, only if cross-compiling from linux/amd64
  if [[ $(kun::golang::host_platform) == "linux/amd64" ]]; then
    # Dynamic CGO linking for other server architectures than linux/amd64 goes here
    # If you want to include support for more server platforms than these, add arch-specific gcc names here
    case "${platform}" in
    "linux/arm")
      export CGO_ENABLED=1
      export CC=arm-linux-gnueabihf-gcc
      ;;
    "linux/arm64")
      export CGO_ENABLED=1
      export CC=aarch64-linux-gnu-gcc
      ;;
    "linux/ppc64le")
      export CGO_ENABLED=1
      export CC=powerpc64le-linux-gnu-gcc
      ;;
    "linux/s390x")
      export CGO_ENABLED=1
      export CC=s390x-linux-gnu-gcc
      ;;
    esac
  fi
}

kun::golang::unset_platform_envs() {
  unset GOOS
  unset GOARCH
  unset GOROOT
  unset CGO_ENABLED
  unset CC
}

# Create the GOPATH tree under $KUN_OUTPUT
kun::golang::create_gopath_tree() {
  local go_pkg_dir="${KUN_GOPATH}/src/${KUN_GO_PACKAGE}"
  local go_pkg_basedir=$(dirname "${go_pkg_dir}")

  mkdir -p "${go_pkg_basedir}"

  # TODO: This symlink should be relative.
  if [[ ! -e "${go_pkg_dir}" || "$(readlink ${go_pkg_dir})" != "${KUN_ROOT}" ]]; then
    ln -snf "${KUN_ROOT}" "${go_pkg_dir}"
  fi

  # create pkg dir
  local go_mod_dir="${KUN_GOPATH}/pkg/mod"
  local go_mod_basedir=$(dirname "${go_mod_dir}")

  mkdir -p "${go_mod_basedir}"

  # 如果系统GOPATH/pkg/mod不存在，则创建
  if [[ -n "${ORIGIN_GOPATH}" && ! -e "${ORIGIN_GOPATH}/pkg/mod" ]]; then
    echo "gopath=${ORIGIN_GOPATH}/pkg/mod"
    mkdir -p "${ORIGIN_GOPATH}/pkg/mod"
  fi

  if [[ ! -e "${go_mod_dir}" && -n "${ORIGIN_GOPATH}" ]]; then
    ln -snf "${ORIGIN_GOPATH}/pkg/mod" "${go_mod_dir}"
  fi

  cat >"${KUN_GOPATH}/BUILD" <<EOF
# This dummy BUILD file prevents Bazel from trying to descend through the
# infinite loop created by the symlink at
# ${go_pkg_dir}
EOF
}

# Ensure the go tool exists and is a viable version.
kun::golang::verify_go_version() {
  if [[ -z "$(which go)" ]]; then
    kun::log::usage_from_stdin <<EOF
Can't find 'go' in PATH, please fix and retry.
See http://golang.org/doc/install for installation instructions.
EOF
    return 2
  fi

  local go_version
  go_version=($(go version))
  local minimum_go_version
  minimum_go_version=go1.10.1
  if [[ "${minimum_go_version}" != $(echo -e "${minimum_go_version}\n${go_version[2]}" | sort -s -t. -k 1,1 -k 2,2n -k 3,3n | head -n1) && "${go_version[2]}" != "devel" ]]; then
    kun::log::usage_from_stdin <<EOF
Detected go version: ${go_version[*]}.
Kubernetes requires ${minimum_go_version} or greater.
Please install ${minimum_go_version} or later.
EOF
    return 2
  fi
}

# kun::golang::setup_env will check that the `go` commands is available in
# ${PATH}. It will also check that the Go version is good enough for the
# Kun build.
#
# Inputs:
#   KUN_EXTRA_GOPATH - If set, this is included in created GOPATH
#
# Outputs:
#   env-var GOPATH points to our local output dir
#   env-var GOBIN is unset (we want binaries in a predictable place)
#   env-var GO15VENDOREXPERIMENT=1
#   current directory is within GOPATH
kun::golang::setup_env() {
  kun::golang::verify_go_version

  kun::golang::create_gopath_tree

  export GOPATH=${KUN_GOPATH}
  export GO111MODULE=on

  # Append KUN_EXTRA_GOPATH to the GOPATH if it is defined.
  if [[ -n ${KUN_EXTRA_GOPATH:-} ]]; then
    GOPATH="${GOPATH}:${KUN_EXTRA_GOPATH}"
  fi

  # Change directories so that we are within the GOPATH.  Some tools get really
  # upset if this is not true.  We use a whole fake GOPATH here to collect the
  # resultant binaries.  Go will not let us use GOBIN with `go install` and
  # cross-compiling, and `go install -o <file>` only works for a single pkg.
  local subdir
  subdir=$(kun::realpath . | sed "s|$KUN_ROOT||")
  cd "${KUN_GOPATH}/src/${KUN_GO_PACKAGE}/${subdir}"

  # Set GOROOT so binaries that parse code can work properly.
  export GOROOT=$(go env GOROOT)

  # Unset GOBIN in case it already exists in the current session.
  unset GOBIN

  # This seems to matter to some tools (godep, ugorji, ginkgo...)
  export GO15VENDOREXPERIMENT=1
}

# This will take binaries from $GOPATH/bin and copy them to the appropriate
# place in ${KUN_OUTPUT_BINDIR}
#
# Ideally this wouldn't be necessary and we could just set GOBIN to
# KUN_OUTPUT_BINDIR but that won't work in the face of cross compilation.  'go
# install' will place binaries that match the host platform directly in $GOBIN
# while placing cross compiled binaries into `platform_arch` subdirs.  This
# complicates pretty much everything else we do around packaging and such.
kun::golang::place_bins() {
  local host_platform
  host_platform=$(kun::golang::host_platform)

  V=2 kun::log::status "Placing binaries"

  local platform
  for platform in "${KUN_CLIENT_PLATFORMS[@]}"; do
    # The substitution on platform_src below will replace all slashes with
    # underscores.  It'll transform darwin/amd64 -> darwin_amd64.
    local platform_src="/${platform//\//_}"
    if [[ $platform == $host_platform ]]; then
      platform_src=""
      rm -f "${THIS_PLATFORM_BIN}"
      ln -s "${KUN_OUTPUT_BINPATH}/${platform}" "${THIS_PLATFORM_BIN}"
    fi

    local full_binpath_src="${KUN_GOPATH}/bin${platform_src}"
    if [[ -d "${full_binpath_src}" ]]; then
      mkdir -p "${KUN_OUTPUT_BINPATH}/${platform}"
      find "${full_binpath_src}" -maxdepth 1 -type f -exec \
        rsync -pc {} "${KUN_OUTPUT_BINPATH}/${platform}" \;
    fi
  done
}

kun::golang::build_binaries_for_platform() {
  local platform=$1
  local use_go_build=${2-}

  local -a statics=()
  local -a nonstatics=()
  local -a tests=()

  V=2 kun::log::info "Env for ${platform}: GOOS=${GOOS-} GOARCH=${GOARCH-} GOROOT=${GOROOT-} CGO_ENABLED=${CGO_ENABLED-} CC=${CC-}"

  for binary in "${binaries[@]}"; do
    if [[ "${binary}" =~ ".test"$ ]]; then
      tests+=($binary)
    elif kun::golang::is_statically_linked_library "${binary}"; then
      statics+=($binary)
    else
      nonstatics+=($binary)
    fi
  done

  if [[ "${#statics[@]}" != 0 ]]; then
    kun::golang::fallback_if_stdlib_not_installable
  fi

  if [[ -n ${use_go_build:-} ]]; then
    kun::log::progress "    "
    for binary in "${statics[@]:+${statics[@]}}"; do
      local outfile=$(kun::golang::output_filename_for_binary "${binary}" "${platform}")
      CGO_ENABLED=0 go build -o "${outfile}" \
        "${goflags[@]:+${goflags[@]}}" \
        -gcflags "${gogcflags}" \
        -ldflags "${goldflags}" \
        "${binary}"
      kun::log::progress "*"
    done
    for binary in "${nonstatics[@]:+${nonstatics[@]}}"; do
      local outfile=$(kun::golang::output_filename_for_binary "${binary}" "${platform}")
      go build -o "${outfile}" \
        "${goflags[@]:+${goflags[@]}}" \
        -gcflags "${gogcflags}" \
        -ldflags "${goldflags}" \
        "${binary}"
      kun::log::progress "*"
    done
    kun::log::progress "\n"
  else
    # Use go install.
    if [[ "${#nonstatics[@]}" != 0 ]]; then
      go install "${goflags[@]:+${goflags[@]}}" \
        -gcflags "${gogcflags}" \
        -ldflags "${goldflags}" \
        "${nonstatics[@]:+${nonstatics[@]}}"
    fi
    if [[ "${#statics[@]}" != 0 ]]; then
      CGO_ENABLED=0 go install -installsuffix cgo "${goflags[@]:+${goflags[@]}}" \
        -gcflags "${gogcflags}" \
        -ldflags "${goldflags}" \
        "${statics[@]:+${statics[@]}}"
    fi
  fi

  for test in "${tests[@]:+${tests[@]}}"; do
    local outfile=$(kun::golang::output_filename_for_binary "${test}" \
      "${platform}")

    local testpkg="$(dirname ${test})"

    # Staleness check always happens on the host machine, so we don't
    # have to locate the `teststale` binaries for the other platforms.
    # Since we place the host binaries in `$KUN_GOPATH/bin`, we can
    # assume that the binary exists there, if it exists at all.
    # Otherwise, something has gone wrong with building the `teststale`
    # binary and we should safely proceed building the test binaries
    # assuming that they are stale. There is no good reason to error
    # out.
    if test -x "${KUN_GOPATH}/bin/teststale" && ! "${KUN_GOPATH}/bin/teststale" -binary "${outfile}" -package "${testpkg}"; then
      continue
    fi

    # `go test -c` below directly builds the binary. It builds the packages,
    # but it never installs them. `go test -i` only installs the dependencies
    # of the test, but not the test package itself. So neither `go test -c`
    # nor `go test -i` installs, for example, test/e2e.a. And without that,
    # doing a staleness check on k8s.io/kubernetes/test/e2e package always
    # returns true (always stale). And that's why we need to install the
    # test package.
    go install "${goflags[@]:+${goflags[@]}}" \
      -gcflags "${gogcflags}" \
      -ldflags "${goldflags}" \
      "${testpkg}"

    mkdir -p "$(dirname ${outfile})"
    go test -i -c \
      "${goflags[@]:+${goflags[@]}}" \
      -gcflags "${gogcflags}" \
      -ldflags "${goldflags}" \
      -o "${outfile}" \
      "${testpkg}"
  done
}

# Build binaries targets specified
#
# Input:
#   $@ - targets and go flags.  If no targets are set then all binaries targets
#     are built.
#   KUN_BUILD_PLATFORMS - Incoming variable of targets to build for.  If unset
#     then just the host architecture is built.
kun::golang::build_binaries() {
  # Create a sub-shell so that we don't pollute the outer environment
  (
    # Check for `go` binary and set ${GOPATH}.
    kun::golang::setup_env
    V=2 kun::log::info "Go version: $(go version)"

    local host_platform
    host_platform=$(kun::golang::host_platform)

    # Use eval to preserve embedded quoted strings.
    local goflags goldflags gogcflags
    eval "goflags=(${GOFLAGS:-})"
    goldflags="${GOLDFLAGS:-} $(kun::version::ldflags)"
    gogcflags="${GOGCFLAGS:-}"

    local use_go_build
    local -a targets=()
    local arg

    for arg; do
      if [[ "${arg}" == "--use_go_build" ]]; then
        use_go_build=true
      elif [[ "${arg}" == -* ]]; then
        # Assume arguments starting with a dash are flags to pass to go.
        goflags+=("${arg}")
      else
        targets+=("${arg}")
      fi
    done

    if [[ ${#targets[@]} -eq 0 ]]; then
      targets=("${KUN_ALL_TARGETS[@]}")
    fi

    local -a platforms=(${KUN_BUILD_PLATFORMS:-})
    if [[ ${#platforms[@]} -eq 0 ]]; then
      platforms=("${host_platform}")
    fi

    # funclet only support linux platforms
    if [[ ! $platforms == linux* ]]; then
      delete=(
        cmd/funclet
      )
      for del in "${delete[@]}"; do
        for i in "${!targets[@]}"; do
          if [[ ${targets[i]} == $del ]]; then
            unset 'targets[i]'
          fi
        done
      done
    fi
    echo "${targets[@]}"

    local binaries
    binaries=($(kun::golang::binaries_from_targets "${targets[@]}"))

    local parallel=false
    if [[ ${#platforms[@]} -gt 1 ]]; then
      local gigs
      gigs=$(kun::golang::get_physmem)

      if [[ ${gigs} -ge ${KUN_PARALLEL_BUILD_MEMORY} ]]; then
        kun::log::status "Multiple platforms requested and available ${gigs}G >= threshold ${KUN_PARALLEL_BUILD_MEMORY}G, building platforms in parallel"
        parallel=true
      else
        kun::log::status "Multiple platforms requested, but available ${gigs}G < threshold ${KUN_PARALLEL_BUILD_MEMORY}G, building platforms in serial"
        parallel=false
      fi
    fi

    # First build the toolchain before building any other targets
    #kun::golang::build_kun_toolchain

    #kun::log::status "Generating bindata:" "${KUN_BINDATAS[@]}"
    #for bindata in ${KUN_BINDATAS[@]}; do
    # Only try to generate bindata if the file exists, since in some cases
    # one-off builds of individual directories may exclude some files.
    #if [[ -f "${KUN_ROOT}/${bindata}" ]]; then
    #go generate "${goflags[@]:+${goflags[@]}}" "${KUN_ROOT}/${bindata}"
    #fi
    #done

    if [[ "${parallel}" == "true" ]]; then
      kun::log::status "Building go targets for {${platforms[*]}} in parallel (output will appear in a burst when complete):" "${targets[@]}"
      local platform
      for platform in "${platforms[@]}"; do
        (
          kun::golang::set_platform_envs "${platform}"
          kun::log::status "${platform}: go build started"
          kun::golang::build_binaries_for_platform ${platform} ${use_go_build:-}
          kun::log::status "${platform}: go build finished"
        ) &>"/tmp//${platform//\//_}.build" &
      done

      local fails=0
      for job in $(jobs -p); do
        wait ${job} || let "fails+=1"
      done

      for platform in "${platforms[@]}"; do
        cat "/tmp//${platform//\//_}.build"
      done

      exit ${fails}
    else
      for platform in "${platforms[@]}"; do
        kun::log::status "Building go targets for ${platform}:" "${targets[@]}"
        (
          kun::golang::set_platform_envs "${platform}"
          kun::golang::build_binaries_for_platform ${platform} ${use_go_build:-}
        )
      done
    fi
  )
}
