FROM alpine:3.22

RUN apk --no-cache add ca-certificates tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

ENV TZ=Asia/Shanghai \
    DOCUMENT_LISTEN_HOST=0.0.0.0 \
    DOCUMENT_DB_PATH=/home/document/data/document.db \
    DOCUMENT_STORAGE_PATH=/home/document/data/objects

WORKDIR /home/document
COPY ./builder/server /home/document/server

EXPOSE 8001
CMD ["/home/document/server", "server:start"]
