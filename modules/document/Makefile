PROJECT_NAME := document
DOCUMENT_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
BUILD_DIR ?= $(DOCUMENT_DIR)/builder
CHART_DIR := $(DOCUMENT_DIR)/charts

IMAGE_REPOSITORY ?= $(shell awk '$$1=="repository:" {print $$2; exit}' $(CHART_DIR)/values.yaml)
IMAGE_TAG ?= $(shell awk '$$1=="appVersion:" {gsub(/"/, "", $$2); print $$2; exit}' $(CHART_DIR)/Chart.yaml)
LOCAL_IMAGE ?= $(PROJECT_NAME):$(IMAGE_TAG)
PUBLISH_IMAGE := $(IMAGE_REPOSITORY):$(IMAGE_TAG)
HELM_CHART_VERSION ?= $(shell awk '$$1=="version:" {print $$2; exit}' $(CHART_DIR)/Chart.yaml)
HELM_APP_VERSION ?= $(IMAGE_TAG)
BETA_SUFFIX ?=
GOOS ?= linux
GOARCH ?= amd64
BUILD_CC ?= x86_64-linux-musl-gcc
BUILD_CXX ?= x86_64-linux-musl-g++

.PHONY: build makebuild dockerbuild ui-deps ui webzip chart lint test publish beta help

build:
	mkdir -p "$(BUILD_DIR)"
	cd "$(DOCUMENT_DIR)" && CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) CC=$(BUILD_CC) CXX=$(BUILD_CXX) go build -trimpath -ldflags "-w -s" -o "$(BUILD_DIR)/server" .

makebuild: build

dockerbuild: build
	docker build --platform linux/$(GOARCH) -f "$(DOCUMENT_DIR)/Dockerfile" -t "$(LOCAL_IMAGE)" "$(DOCUMENT_DIR)"

ui-deps:
	cd "$(DOCUMENT_DIR)/ui" && npm ci

ui:
	cd "$(DOCUMENT_DIR)/ui" && npm run build -- --dest "$(BUILD_DIR)/ui"

webzip: ui
	mkdir -p "$(BUILD_DIR)"
	@archive_dir=$$(mktemp -d); \
	trap 'rm -rf "$$archive_dir"' EXIT; \
	cd "$(BUILD_DIR)/ui" && zip -qr "$$archive_dir/document-ui.zip" . && \
	mv "$$archive_dir/document-ui.zip" "$(BUILD_DIR)/document-ui.zip"

chart:
	mkdir -p "$(BUILD_DIR)"
	@chart_dir=$$(mktemp -d); \
	trap 'rm -rf "$$chart_dir"' EXIT; \
	cp -R "$(CHART_DIR)" "$$chart_dir/document"; \
	awk -v repository="$(IMAGE_REPOSITORY)" -v tag="$(IMAGE_TAG)" ' \
		/^image:/ { in_image=1; print; next } \
		in_image && /^[^[:space:]]/ { in_image=0 } \
		in_image && /^[[:space:]]*repository:/ { sub(/repository:.*/, "repository: " repository) } \
		in_image && /^[[:space:]]*tag:/ { sub(/tag:.*/, "tag: " tag) } \
		{ print } \
	' "$$chart_dir/document/values.yaml" > "$$chart_dir/values.yaml" && \
	mv "$$chart_dir/values.yaml" "$$chart_dir/document/values.yaml" && \
	helm package "$$chart_dir/document" --destination "$(BUILD_DIR)" --version "$(HELM_CHART_VERSION)" --app-version "$(HELM_APP_VERSION)"

lint:
	@test -n "$(CHART_PACKAGE)" || (echo '请指定平台注入 Sidecar 模板后的 CHART_PACKAGE'; exit 1)
	helm lint "$(CHART_PACKAGE)"
	helm template document "$(CHART_PACKAGE)" > /dev/null

test:
	cd "$(DOCUMENT_DIR)" && go test . ./app/... ./common/...

publish: dockerbuild webzip chart
	docker tag "$(LOCAL_IMAGE)" "$(PUBLISH_IMAGE)"
	docker push "$(PUBLISH_IMAGE)"

beta:
	@test -n "$(BETA_SUFFIX)" || (echo '请指定 BETA_SUFFIX，例如 beta1'; exit 1)
	$(MAKE) -C "$(DOCUMENT_DIR)" publish IMAGE_TAG="$(IMAGE_TAG)-$(BETA_SUFFIX)" HELM_APP_VERSION="$(IMAGE_TAG)-$(BETA_SUFFIX)" BUILD_DIR="$(BUILD_DIR)"

help:
	@echo 'make build / dockerbuild：编译 Linux musl 二进制 / 构建镜像'
	@echo 'make ui-deps / webzip：安装前端依赖 / 构建前端 ZIP'
	@echo 'make chart / lint / test：打包 Chart / 检查模板 / 测试文档模块'
	@echo 'make publish / beta BETA_SUFFIX=beta1：构建产物并推送镜像'
	@echo '产物默认输出至 builder；支持 BUILD_DIR、IMAGE_REPOSITORY、IMAGE_TAG、HELM_CHART_VERSION'
