FROM alpine

RUN apk --no-cache add zip tzdata

RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
      echo "Asia/Shanghai" > /etc/timezone

ENV TZ=Asia/Shanghai

COPY ./icon.png /home/icon.png
COPY ./builder/server /home/rangine
COPY ./config.yaml /home/config.yaml
COPY ./w7-cd-artifact-online.sql /home/w7-cd-artifact-online.sql

RUN mkdir -p /home/zpk/db/ /home/zpk/cert/

EXPOSE 8000

CMD ["/home/rangine", "server:start", "-f", "/home/config.yaml"]