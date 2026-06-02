PROJECT_NAME := rangine

GO_BASE := $(shell pwd)
GO_BIN := $(GO_BASE)/bin
FILE_NAME := $(shell date +%Y%m%d%H%M)
SOURCE_FILES := *.go

HELM_VALUES_FILE := charts/values.yaml
HELM_IMAGE_REPOSITORY := $(shell awk '/^image:/{flag=1; next} flag && /^[^[:space:]]/{flag=0} flag && $$1=="repository:" {print $$2; exit}' $(HELM_VALUES_FILE))
HELM_IMAGE_TAG := $(shell awk '/^image:/{flag=1; next} flag && /^[^[:space:]]/{flag=0} flag && $$1=="tag:" {print $$2; exit}' $(HELM_VALUES_FILE))
HELM_CHART_VERSION ?= $(shell awk '$$1=="version:" {print $$2; exit}' charts/Chart.yaml)
HELM_APP_VERSION ?= $(IMAGE_TAG)
BETA_SUFFIX ?=
BETA_IMAGE_TAG ?= $(IMAGE_TAG)-$(BETA_SUFFIX)

OFFICIAL_RELEASE ?= false
OFFICIAL_IMAGE_REPOSITORY ?= ccr.ccs.tencentyun.com/w7team/zpk
OFFICIAL_IMAGE_TAG ?=
OFFICIAL_IMAGE ?= $(OFFICIAL_IMAGE_REPOSITORY):$(OFFICIAL_IMAGE_TAG)
IMAGE_REPOSITORY ?= $(HELM_IMAGE_REPOSITORY)
IMAGE_TAG ?= $(if $(filter true,$(OFFICIAL_RELEASE)),$(OFFICIAL_IMAGE_TAG),$(HELM_IMAGE_TAG))
LOCAL_IMAGE ?= $(PROJECT_NAME):$(IMAGE_TAG)
PUBLISH_IMAGE ?= $(if $(filter true,$(OFFICIAL_RELEASE)),$(OFFICIAL_IMAGE),$(IMAGE_REPOSITORY):$(IMAGE_TAG))
HELM_PACKAGE_IMAGE_REPOSITORY ?= $(if $(filter true,$(OFFICIAL_RELEASE)),$(OFFICIAL_IMAGE_REPOSITORY),$(IMAGE_REPOSITORY))
HELM_PACKAGE_IMAGE_TAG ?= $(if $(filter true,$(OFFICIAL_RELEASE)),$(OFFICIAL_IMAGE_TAG),$(IMAGE_TAG))

.PHONY: tidy build makebuild dockerbuild validate-publish publish beta dev test clean help

tidy:
	go mod tidy

build: clean tidy
	CGO_ENABLED=1 GOARCH=amd64 GOOS=linux CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ go build -gcflags=-trimpath=$$GOPATH -asmflags=-trimpath=$$GOPATH -ldflags "-w -s" -o builder/server ${SOURCE_FILES}

makebuild: build

dockerbuild:
	docker build -t $(LOCAL_IMAGE) .

validate-publish:
	@if [ "$(OFFICIAL_RELEASE)" = "true" ] && [ -z "$(OFFICIAL_IMAGE_TAG)" ]; then \
		echo "OFFICIAL_IMAGE_TAG is required when OFFICIAL_RELEASE=true"; \
		exit 1; \
	fi

publish: validate-publish makebuild dockerbuild
	docker tag $(LOCAL_IMAGE) $(PUBLISH_IMAGE)
	docker push $(PUBLISH_IMAGE)
	rm -f charts/zpk-*.tgz
	tmp_chart_dir=$$(mktemp -d); \
	trap 'rm -rf "$$tmp_chart_dir"' EXIT; \
	cp -R charts "$$tmp_chart_dir/zpk"; \
	rm -f "$$tmp_chart_dir"/zpk/zpk-*.tgz; \
	awk -v repository="$(HELM_PACKAGE_IMAGE_REPOSITORY)" -v tag="$(HELM_PACKAGE_IMAGE_TAG)" ' \
		/^image:[[:space:]]*$$/ { in_image=1; print; next } \
		in_image && /^[^[:space:]]/ { in_image=0 } \
		in_image && /^[[:space:]]*repository:/ { sub(/repository:.*/, "repository: " repository) } \
		in_image && /^[[:space:]]*tag:/ { sub(/tag:.*/, "tag: " tag) } \
		{ print } \
	' "$$tmp_chart_dir/zpk/values.yaml" > "$$tmp_chart_dir/zpk/values.yaml.tmp"; \
	mv "$$tmp_chart_dir/zpk/values.yaml.tmp" "$$tmp_chart_dir/zpk/values.yaml"; \
	helm package "$$tmp_chart_dir/zpk" -d charts --version $(HELM_CHART_VERSION) --app-version $(HELM_APP_VERSION)

beta:
	@if [ -z "$(BETA_SUFFIX)" ]; then \
		echo "BETA_SUFFIX is required, for example: make beta BETA_SUFFIX=beta1"; \
		exit 1; \
	fi
	$(MAKE) publish IMAGE_TAG=$(BETA_IMAGE_TAG) HELM_APP_VERSION=$(BETA_IMAGE_TAG)

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
	@echo "make tidy - 整理 Go module 依赖"
	@echo "make makebuild - 执行现有构建流程"
	@echo "make dockerbuild - 构建本地镜像 LOCAL_IMAGE=$(LOCAL_IMAGE)"
	@echo "make publish - 构建二进制、镜像、打 tag、push，并重新打 helm/zpk tgz"
	@echo "make beta BETA_SUFFIX=beta1 - 使用当前镜像 tag 加手动后缀发布 beta，例如 $(IMAGE_TAG)-beta1"
	@echo "官方发布: OFFICIAL_RELEASE=true OFFICIAL_IMAGE_TAG=xxx"
	@echo "可覆盖变量: IMAGE_REPOSITORY=xxx IMAGE_TAG=xxx HELM_CHART_VERSION=xxx HELM_APP_VERSION=xxx HELM_PACKAGE_IMAGE_REPOSITORY=xxx HELM_PACKAGE_IMAGE_TAG=xxx LOCAL_IMAGE=xxx PUBLISH_IMAGE=xxx OFFICIAL_IMAGE_REPOSITORY=xxx"
