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

# Alpine 自带生命周期脚本所需的 BusyBox wget/unzip 和 CA 证书；构建时显式校验，避免运行时才发现缺失。
RUN command -v wget >/dev/null \
    && command -v unzip >/dev/null \
    && test -s /etc/ssl/certs/ca-certificates.crt

COPY --from=builder /out/w7-tradition-plugin /usr/local/bin/w7-tradition-plugin

ENTRYPOINT ["w7-tradition-plugin"]
CMD ["--help"]
