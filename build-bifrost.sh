#!/bin/bash
#
# This script will build bindings for OSX and Linux and check it into the public repo
# (You still have to merge it)
#
COLOR_RESET="\033[0m"
COLOR_RED="\033[38;5;9m"
COLOR_LIGHTCYAN="\033[1;36m"
COLOR_LIGHTGREEN="\033[1;32m"

COMMANDS=(bifrost)
OUT_DIR=${OUT_DIR:-./}
BUILD_TIME=`date | sed -e 's/ /_/g'`
TARGET_OS=${TARGET_OS:-darwin}
TARGET_ARCH=${TARGET_ARCH:-amd64}

GIT_HASH=${GIT_HASH:-$(git rev-parse --short HEAD)}
VERSION=v${MAJOR_VER}.${MINOR_VER}

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

check_prerequisites() {
    if [ "${LOCAL_BUILD}" = "1" ] ; then
        # we're inside the bifrost container. no need to check docker daemon
        return
    fi
    docker ps >/dev/null 2>&1
    if [ $? -gt 0 ] ; then
        error "Docker failed to respond! Please check if Docker Engine is running"
    fi
}

build() {
    local CMD=$1
    info "Building ${CMD} for ${TARGET_OS} ${TARGET_ARCH}..."

    if [[ "$TARGET_OS" = "darwin" || "$TARGET_OS" = "linux" ]] ; then
        local OUTPUT_FILE="$CMD"
    else
        local OUTPUT_FILE="${CMD}.exe"
    fi
    local OUTPUT_PATH="${OUT_DIR}/${OUTPUT_FILE}"

    # build
    go build -ldflags "-X main.BuildTime=${BUILD_TIME} -X main.Version=${VERSION}-${GIT_HASH}" \
             -o $OUTPUT_PATH bifrost.go sample_services.go sample_vm_service.go
    if [ $? -ne 0 ] ; then
        error "Build Failed!"
    fi

    chmod +x ${OUTPUT_PATH}
#
#    # call the binary only if target OS and current OS match
#    if [[ "$(uname -s)" = "Darwin" && $TARGET_OS = "darwin" ]] ; then
#        $OUT_DIR/$CMD --version
#    fi
#
#    if [[ "$(uname -s)" = "Linux" && $TARGET_OS = "linux" ]] ; then
#        $OUT_DIR/$CMD --version
#    fi
}

trap '_trap' SIGINT SIGTERM

while getopts ":o:a:" flag ; do
    case $flag in
        o)
            TARGET_OS=${OPTARG}
            ;;
        a)
            TARGET_ARCH=${OPTARG}
            ;;
        *)
            echo "Usage: $0 [-a architecture - 386, amd64] [-o target OS - windows, darwin, linux]" >&2
            exit 1
            ;;
    esac
done

# ensure output dir exist
mkdir -p ${OUT_DIR}

check_prerequisites
for CMD in ${COMMANDS[@]} ; do
    GOOS=${TARGET_OS} GOARCH=${TARGET_ARCH} build $CMD
done

