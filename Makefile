.PHONY: build debug release test benchmark clean dev

# 平台
# windows or linux
OS := linux

# 服务名称
PROJECT := gateway
CUR_PWD := $(shell pwd)

# 输出目录
OUTPUT_PATH := $(CUR_PWD)/__output
OUTPUT_RELEASE_PATH := $(CUR_PWD)/__output/release
OUTPUT_DEBUG_PATH := $(CUR_PWD)/__output/debug
OUTPUT_DEV_PATH := $(CUR_PWD)/__output/dev

# 二进制的路径和名称
APP_PATH := $(CUR_PWD)/cmd/gateway
ifeq ($(OS),windows)
# win32
APP_BIN := app.exe
else
# linux
APP_BIN := gateway
endif

# 给二进制文件注入构建参数
AUTHOR := $(shell git log --pretty=format:"%an"|head -n 1)
VERSION := $(shell git rev-list HEAD | head -1)
BUILD_INFO := $(shell git log --pretty=format:"%s" | head -1)
BUILD_DATE := $(shell date +%Y-%m-%d\ %H:%M:%S)

LD_FLAGS=-X "$(PROJECT)/pkg/version.VERSION=$(VERSION)" -X "$(PROJECT)/pkg/version.AUTHOR=$(AUTHOR)" -X "$(PROJECT)/pkg/version.BUILD_INFO=$(BUILD_INFO)" -X "$(PROJECT)/pkg/version.BUILD_DATE=$(BUILD_DATE)"

# 测试文件
TEST_FILES := ""

# 默认为dev
default: dev

# go的环境变量
export GO111MODULE=on
export GOPROXY=https://goproxy.cn,direct
export CGO_ENABLED=0
export GOOS=$(OS)
export GOARCH=amd64


# 调试版本（开发环境）
dev: test
	@echo "build binaries: $(OS) dev"
	mkdir -p $(OUTPUT_DEV_PATH)
	go mod download; go build  -ldflags '-X "$(PROJECT)/pkg/version.ENV=dev" $(LD_FLAGS)' -gcflags "-N -l"  -o $(OUTPUT_DEV_PATH)/$(APP_BIN) $(APP_PATH)


# 调试版本（测试环境）
debug: test
	@echo "build binaries: $(OS) debug"
	mkdir -p $(OUTPUT_DEBUG_PATH)
	go mod download; go build  -ldflags '-X "$(PROJECT)/pkg/version.ENV=debug" $(LD_FLAGS)' -gcflags "-N -l"  -o $(OUTPUT_DEBUG_PATH)/$(APP_BIN) $(APP_PATH)

# 发布版本（正式环境）
release: test
	@echo "build binaries: $(OS) release"
	mkdir -p $(OUTPUT_RELEASE_PATH)
	go mod download; go build  -ldflags '-X "$(PROJECT)/pkg/version.ENV=release" $(LD_FLAGS)' -o $(OUTPUT_RELEASE_PATH)/$(APP_BIN) $(APP_PATH)

# 单元测试
test:
	@echo "run tests"

# 压力测试
benchmark:
	@echo "run benchmarks"

# 清理
clean:
	rm -rf $(OUTPUT_PATH)
