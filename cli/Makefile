CLI_NAME := w7-zpk
CLI_DIR := $(shell pwd)
BIN_DIR := $(CLI_DIR)/bin
GO_BUILD := CGO_ENABLED=0 go build

.PHONY: build build-all build-darwin build-linux build-windows clean help

build: build-all

build-all: build-darwin build-linux build-windows

build-darwin:
	mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR)/$(CLI_NAME)_darwin_amd64 .

build-linux:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR)/$(CLI_NAME)_linux_amd64 .

build-windows:
	mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR)/$(CLI_NAME)_windows_amd64.exe .

clean:
	rm -rf $(BIN_DIR)

help:
	@echo "make build - 构建全部 CLI 可执行文件"
	@echo "make build-darwin - 构建 macOS CLI"
	@echo "make build-linux - 构建 Linux CLI"
	@echo "make build-windows - 构建 Windows CLI"
	@echo "make clean - 删除 cli/bin"
