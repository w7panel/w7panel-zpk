PROJECT_NAME := rangine

GO_BASE := $(shell pwd)
GO_BIN := $(GO_BASE)/bin
FILE_NAME := $(shell date +%Y%m%d%H%M)
SOURCE_FILES := *.go

HELM_VALUES_FILE := charts/values.yaml
HELM_IMAGE_REPOSITORY := $(shell awk '/^image:/{flag=1; next} flag && /^[^[:space:]]/{flag=0} flag && $$1=="repository:" {print $$2; exit}' $(HELM_VALUES_FILE))
HELM_IMAGE_TAG := $(shell awk '/^image:/{flag=1; next} flag && /^[^[:space:]]/{flag=0} flag && $$1=="tag:" {print $$2; exit}' $(HELM_VALUES_FILE))

IMAGE_REPOSITORY ?= $(HELM_IMAGE_REPOSITORY)
IMAGE_TAG ?= $(HELM_IMAGE_TAG)
LOCAL_IMAGE ?= $(PROJECT_NAME):$(IMAGE_TAG)
PUBLISH_IMAGE ?= $(IMAGE_REPOSITORY):$(IMAGE_TAG)
OFFICIAL_RELEASE ?= false
OFFICIAL_IMAGE_REPOSITORY ?= ccr.ccs.tencentyun.com/w7team/zpk
OFFICIAL_IMAGE_TAG ?=
OFFICIAL_IMAGE ?= $(OFFICIAL_IMAGE_REPOSITORY):$(OFFICIAL_IMAGE_TAG)

.PHONY: build-osx build build-windows makebuild dockerbuild publish dev test clean help

build-osx: clean
	go build -o ${GO_BIN}/${PROJECT_NAME}_osx ${SOURCE_FILES}

build: clean
	CGO_ENABLED=1 GOARCH=amd64 GOOS=linux CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ go build -gcflags=-trimpath=$$GOPATH -asmflags=-trimpath=$$GOPATH -ldflags "-w -s" -o builder/server ${SOURCE_FILES}

makebuild: build

build-windows: clean
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ${GO_BIN}/${PROJECT_NAME}.exe ${SOURCE_FILES}

dockerbuild:
	docker build -t $(LOCAL_IMAGE) .

publish: makebuild dockerbuild
	docker tag $(LOCAL_IMAGE) $(PUBLISH_IMAGE)
	docker push $(PUBLISH_IMAGE)
	@if [ "$(OFFICIAL_RELEASE)" = "true" ]; then \
		if [ -z "$(OFFICIAL_IMAGE_TAG)" ]; then \
			echo "OFFICIAL_IMAGE_TAG is required when OFFICIAL_RELEASE=true"; \
			exit 1; \
		fi; \
		docker tag $(LOCAL_IMAGE) $(OFFICIAL_IMAGE); \
		docker push $(OFFICIAL_IMAGE); \
	fi
	rm -f charts/zpk-*.tgz
	helm package charts/ -d charts

dev: clean
	go run ${SOURCE_FILES} server:start

test: clean
	go test -v ./tests/...

clean:
	go clean
	rm -rf ${GO_BIN}/*
	rm -rf ./output/*
	rm -rf w7_cd_artifact.zip runtime/w7_cd_artifact

help:
	@echo "make build - 编译 Linux 二进制到 builder/server"
	@echo "make makebuild - 执行现有构建流程"
	@echo "make dockerbuild - 构建本地镜像 LOCAL_IMAGE=$(LOCAL_IMAGE)"
	@echo "make publish - 构建二进制、镜像、打 tag、push，并重新打 helm/zpk tgz"
	@echo "官方发布: OFFICIAL_RELEASE=true OFFICIAL_IMAGE_TAG=xxx"
	@echo "可覆盖变量: IMAGE_REPOSITORY=xxx IMAGE_TAG=xxx LOCAL_IMAGE=xxx PUBLISH_IMAGE=xxx OFFICIAL_IMAGE_REPOSITORY=xxx"
