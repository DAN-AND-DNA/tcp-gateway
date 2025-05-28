#!/bin/bash

set -ex

CUR_PWD=$(dirname "$(realpath "$0")")
PROJECT='gateway'

APP_ENV=${1:-"dev"}

# 输出目录
OUTPUT_PATH="${CUR_PWD}/__output"
OUTPUT_RELEASE_PATH="${CUR_PWD}/__output/release"
OUTPUT_DEBUG_PATH="${CUR_PWD}/__output/debug"
OUTPUT_DEV_PATH="${CUR_PWD}/__output/dev"
OS=linux
APP_PATH="${CUR_PWD}/cmd/gateway"
APP_BIN="gateway"

cd ${CUR_PWD}

AUTHOR=$(git log --pretty=format:"%an"|head -n 1)
VERSION=$(git rev-list HEAD | head -1)
BUILD_INFO=$(git log --pretty=format:"%s" | head -1)
BUILD_DATE=$(date "+%Y-%m-%d_%H:%M:%S")

#LD_FLAGS="-X ${PROJECT}/pkg/version.VERSION=${VERSION} -X ${PROJECT}/pkg/version.AUTHOR=${AUTHOR} -X ${PROJECT}/pkg/version.BUILD_INFO=${BUILD_INFO} -X ${PROJECT}/pkg/version.BUILD_DATE=${BUILD_DATE}"
#echo $LD_FLAGS

# go的环境变量
export GO111MODULE=on
export GOPROXY=https://goproxy.cn,direct
export CGO_ENABLED=0
export GOOS=$OS
export GOARCH=amd64

OUTPUT_TARGET_PATH=$OUTPUT_DEV_PATH
echo "build binaries: ${OS} ${APP_ENV}"
if [ "$APP_ENV" = "dev" ]; then
    OUTPUT_TARGET_PATH=${OUTPUT_DEV_PATH}
elif [ "$APP_ENV" = "debug" ]; then
    OUTPUT_TARGET_PATH=${OUTPUT_DEBUG_PATH}
elif [ "$APP_ENV" = "release" ]; then
    OUTPUT_TARGET_PATH=${OUTPUT_RELEASE_PATH}
else
    echo "bad app env"
    exit 1
fi

mkdir -p ${OUTPUT_TARGET_PATH}
go mod tidy;go build -o ${OUTPUT_TARGET_PATH}/${APP_BIN} -ldflags "-X ${PROJECT}/pkg/version.ENV=${APP_ENV} -X ${PROJECT}/pkg/version.VERSION=${VERSION} -X ${PROJECT}/pkg/version.AUTHOR=${AUTHOR} -X ${PROJECT}/pkg/version.BUILD_DATE=${BUILD_DATE}" ${APP_PATH}
cp restart.sh ${OUTPUT_TARGET_PATH}/