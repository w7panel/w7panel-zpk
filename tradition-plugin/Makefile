CLI_NAME := w7-tradition-plugin
CLI_DIR := $(shell pwd)
BIN_DIR := $(CLI_DIR)/bin
GO_BUILD := CGO_ENABLED=0 go build -trimpath -ldflags "-s -w"

.PHONY: tidy test build build-all build-darwin build-linux clean help

tidy:
	go mod tidy

test:
	go test ./...

build:
	mkdir -p $(BIN_DIR)
	$(GO_BUILD) -o $(BIN_DIR)/$(CLI_NAME) .

build-all: build-darwin build-linux

build-darwin:
	mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR)/$(CLI_NAME)_darwin_amd64 .
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -o $(BIN_DIR)/$(CLI_NAME)_darwin_arm64 .

build-linux:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR)/$(CLI_NAME)_linux_amd64 .
	GOOS=linux GOARCH=arm64 $(GO_BUILD) -o $(BIN_DIR)/$(CLI_NAME)_linux_arm64 .

clean:
	rm -rf $(BIN_DIR)

help:
	@echo "make test - 运行测试"
	@echo "make build - 构建当前平台可执行文件"
	@echo "make build-all - 构建 macOS/Linux amd64/arm64 可执行文件"
	@echo "make tidy - 整理 Go module 依赖"
	@echo "make clean - 删除构建产物"
