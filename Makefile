# 定义项目名称
BINARY_NAME=xfirefly

# 定义输出目录
OUTPUT_DIR=bin

VERSION    = $(shell git tag --sort=-creatordate | head -n 1)
GIT_COMMIT = $(shell git rev-parse --short HEAD)
BUILD_TIME = $(shell date "+%F")

define LDFLAGS
"-X 'xfirefly/pkg/cli.defaultVersion=${VERSION}' \
 -X 'xfirefly/pkg/cli.defaultGitCommit=${GIT_COMMIT}' \
 -X 'xfirefly/pkg/cli.defaultBuildDate=${BUILD_TIME}'"
endef

.PHONY: build
build:
	go build -ldflags=${LDFLAGS} -o ${OUTPUT_DIR}/${BINARY_NAME}.exe xfirefly.go