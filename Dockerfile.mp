FROM --platform=$BUILDPLATFORM 	hub.kce.ksyun.com/ksyun/golang:1.19.3-alpine AS build
ARG TARGETOS TARGETARCH
RUN mkdir /csi-plugin
ENV GOPROXY=https://goproxy.cn
COPY . /csi-plugin/
WORKDIR /csi-plugin
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o ./bin/csi-diskplugin ./cmd/diskplugin

FROM hub.kce.ksyun.com/ksyun/alpine:3.18.4
#RUN sed -i 's/http\:\/\/dl-cdn.alpinelinux.org/https\:\/\/mirrors.aliyun.com/g' /etc/apk/repositories
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

RUN apk update && apk upgrade && \
    apk add e2fsprogs-extra &&\
    apk add e2fsprogs && apk add blkid && apk add findmnt && \
    apk add xfsprogs && apk add xfsprogs-extra && \
    apk add nfs-utils && \
    apk -U add ca-certificates && \
    #apk add cloud-utils-growpart && \
    apk add util-linux && \
    apk add eudev && \
    update-ca-certificates

RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata

COPY --from=build /csi-plugin/bin/csi-diskplugin /csi-diskplugin
ENTRYPOINT ["/csi-diskplugin"]