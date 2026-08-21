FROM golang:1.25-alpine AS build
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/master ./cmd/master \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/target ./cmd/target

FROM alpine:3.21 AS runtime
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata wget
ENV TZ=Asia/Shanghai

FROM runtime AS master
COPY --from=build /out/master /usr/local/bin/master
EXPOSE 8080 9090
ENTRYPOINT ["/usr/local/bin/master"]

FROM runtime AS worker
COPY --from=build /out/worker /usr/local/bin/worker
ENTRYPOINT ["/usr/local/bin/worker"]

FROM runtime AS target
COPY --from=build /out/target /usr/local/bin/target
EXPOSE 8088
ENTRYPOINT ["/usr/local/bin/target"]
