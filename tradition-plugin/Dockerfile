# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH
ARG GOPROXY=https://goproxy.cn,direct

WORKDIR /src

COPY go.mod go.sum ./
RUN GOPROXY="${GOPROXY}" go mod download

COPY main.go config.yaml ./
COPY app ./app

RUN CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
    go build -trimpath -ldflags="-s -w" -o /out/w7-tradition-plugin .

FROM alpine:3.22

# 优先使用 curl 下载代码包；保留 BusyBox wget 以兼容旧版安装脚本。
RUN apk add --no-cache curl ca-certificates \
    && command -v curl >/dev/null \
    && command -v wget >/dev/null \
    && command -v unzip >/dev/null \
    && test -s /etc/ssl/certs/ca-certificates.crt

COPY --from=builder /out/w7-tradition-plugin /usr/local/bin/w7-tradition-plugin

ENTRYPOINT ["w7-tradition-plugin"]
CMD ["--help"]
