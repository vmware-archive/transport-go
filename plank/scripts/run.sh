#!/bin/bash

ROOT=$(cd $(dirname $0)/.. ; pwd)
HASH="${CI_COMMIT_SHORT_SHA:-$(git branch --no-color | grep '*' | sed 's/\*\ //')-$(git log -1 --format=%h)}"
SHORT_VER="$(cat $ROOT/.version.txt)"
LONG_VER="$SHORT_VER-$HASH"

go run -ldflags "-s -w -X main.version=${LONG_VER}" $ROOT/cmd/main.go $@
