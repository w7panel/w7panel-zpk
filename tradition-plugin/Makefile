CLI_NAME := w7-tradition-plugin
CLI_DIR := $(shell pwd)
BIN_DIR := $(CLI_DIR)/bin
GO_BUILD := CGO_ENABLED=0 go build -trimpath -ldflags "-s -w"
IMAGE_REPOSITORY ?= zpk.w7.cc/public/tradition-plugin
IMAGE_TAG ?= v1.0.0
IMAGE ?= $(IMAGE_REPOSITORY):$(IMAGE_TAG)
PLATFORMS ?= linux/amd64,linux/arm64
GOPROXY ?= https://goproxy.cn,direct

.PHONY: tidy test build build-all build-darwin build-linux image-build image-push image-push-multi clean help

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

image-build:
	docker build --build-arg GOPROXY=$(GOPROXY) -f $(CLI_DIR)/Dockerfile -t $(IMAGE) $(CLI_DIR)

image-push: image-build
	docker push $(IMAGE)

image-push-multi:
	docker buildx build --platform $(PLATFORMS) --build-arg GOPROXY=$(GOPROXY) -f $(CLI_DIR)/Dockerfile -t $(IMAGE) --push $(CLI_DIR)

clean:
	rm -rf $(BIN_DIR)

help:
	@echo "make test - 运行测试"
	@echo "make build - 构建当前平台可执行文件"
	@echo "make build-all - 构建 macOS/Linux amd64/arm64 可执行文件"
	@echo "make image-build - 构建镜像 $(IMAGE)"
	@echo "make image-push - 构建并推送当前平台镜像 $(IMAGE)"
	@echo "make image-push-multi - 构建并推送 $(PLATFORMS) 多平台镜像 $(IMAGE)"
	@echo "make tidy - 整理 Go module 依赖"
	@echo "make clean - 删除构建产物"
