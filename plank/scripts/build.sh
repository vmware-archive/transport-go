#!/bin/bash
# Copyright 2019 VMware, Inc. All Rights Reserved.
#

COLOR_RESET="\033[0m"
COLOR_RED="\033[38;5;9m"
COLOR_LIGHTCYAN="\033[1;36m"
COLOR_LIGHTGREEN="\033[1;32m"

ROOT=$(cd $(dirname $0)/.. ; pwd)
PWD=$(pwd)
DEBUG=${DEBUG:-1}

[ ! -z $ENV ] || ENV=local
ENV=${ENV/prod*/production}

error() {
    echo -e "${COLOR_RED}ERROR: $1${COLOR_RESET}" >&2
    exit 1
}

warn() {
    echo -e "${COLOR_RED}WARNING: $1${COLOR_RESET}"
}

info() {
    echo -e "${COLOR_LIGHTCYAN}$1${COLOR_RESET}"
}

success() {
    echo -e "${COLOR_LIGHTGREEN}$1${COLOR_RESET}"
}

_trap() {
  echo interrupted >&2
  exit 1
}

trap '_trap' SIGINT SIGTERM

# prepare a snippet containing branch (if applicable) and commit hash associated with the current build
# note that any detached branches will have their name dropped in the finalized version string
CURRENT_BRANCH="$(git branch --no-color | grep '*' | sed 's/\*\ //')"
HASH=${HASH:-${CURRENT_BRANCH}}
if [ ! -z "$(echo ${HASH} | grep -oE "(detached|no branch)")" ] ; then
    HASH="$(git log -1 --format=%h)"
else
    HASH="${HASH}-$(git log -1 --format=%h)"
fi

# get the latest tag version
TAGS=($(git tag -l | sed -E 's/^v//g' | sort -t. -k 1,1n -k 2,2n -k 3,3n))
if [ ${#TAGS[@]} -eq 0 ] ; then
    SHORT_VER=""
else
    LAST_TAG_IDX="$[${#TAGS[@]} - 1]"
    SHORT_VER="v${TAGS[${LAST_TAG_IDX}]}"
fi

# only prepend version tag to the version string on main branch or tag object.
# otherwise the version string will be a branch name followed by a commit hash for non-main branches.
IS_TAG_BUILD=$([[ -n "${CI_COMMIT_TAG}" || $(echo ${CURRENT_BRANCH} | grep -o "at ${SHORT_VER}") ]] && echo 1)
IS_MAIN=$([[ "${CI_COMMIT_BRANCH}" = "main" || "${HASH}" =~ "main" ]] && echo 1)
if [[ "${IS_MAIN}" || "${IS_TAG_BUILD}" ]] ; then
    LONG_VER="$SHORT_VER-$HASH"
else
    LONG_VER="$HASH"
fi

BUILD_DIR=${BUILD_DIR:-${ROOT}/build}
OUT_FILENAME="plank"
OUT_FILEEXT=""
export GOOS=${GOOS:-$(uname -s | tr '[:upper:]' '[:lower:]')}

info "Building Plank..."
info "Version: $LONG_VER"
info "Target platform: $GOOS"

if [ ${GOOS} = "windows" ] ; then
    OUT_FILEEXT=".exe"
fi

go generate
go build -ldflags "-s -w -X main.version=${LONG_VER}" -o ${BUILD_DIR}/${OUT_FILENAME}${OUT_FILEEXT} ${ROOT}/cmd/main.go

if [ $? -gt 0 ] ; then
    error "Build failed!"
fi

success "Build succeeded"
