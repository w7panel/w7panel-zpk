CLI_NAME := w7-zpk
CLI_DIR := $(shell pwd)
BIN_DIR := $(CLI_DIR)/bin
GO_BUILD := CGO_ENABLED=0 go build -trimpath -ldflags "-s -w"
UPX := upx
UPX_FLAGS := --best --lzma
UPX_DARWIN_FLAGS := $(UPX_FLAGS) --force-macos

.PHONY: tidy build build-all build-compress build-darwin build-linux build-windows compress clean help

build: build-all

tidy:
	go mod tidy

build-all: tidy build-darwin build-linux build-windows

build-compress: build-all compress

build-darwin:
	mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR)/$(CLI_NAME)_darwin_amd64 .

build-linux:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR)/$(CLI_NAME)_linux_amd64 .

build-windows:
	mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o $(BIN_DIR)/$(CLI_NAME)_windows_amd64.exe .

compress:
	@if ! command -v $(UPX) >/dev/null 2>&1; then \
		echo "upx not found, please install upx first"; \
		exit 1; \
	fi
	$(UPX) $(UPX_DARWIN_FLAGS) $(BIN_DIR)/$(CLI_NAME)_darwin_amd64
	$(UPX) $(UPX_FLAGS) $(BIN_DIR)/$(CLI_NAME)_linux_amd64
	$(UPX) $(UPX_FLAGS) $(BIN_DIR)/$(CLI_NAME)_windows_amd64.exe

clean:
	rm -rf $(BIN_DIR)

help:
	@echo "make build - 构建全部 CLI 可执行文件"
	@echo "make tidy - 整理 Go module 依赖"
	@echo "make build-compress - 构建全部 CLI 可执行文件并使用 UPX 压缩"
	@echo "make build-darwin - 构建 macOS CLI"
	@echo "make build-linux - 构建 Linux CLI"
	@echo "make build-windows - 构建 Windows CLI"
	@echo "make compress - 使用 UPX 压缩 cli/bin 下的构建产物"
	@echo "make clean - 删除 cli/bin"
