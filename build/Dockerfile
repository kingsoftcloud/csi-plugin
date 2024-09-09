FROM golang:1.18.7 as builder
RUN mkdir /csi-plugin
COPY . /csi-plugin/
ENV GOPROXY https://goproxy.cn
WORKDIR /csi-plugin
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o ./bin/csi-diskplugin ./cmd/diskplugin

FROM debian:buster
COPY --from=builder /csi-plugin/bin/csi-diskplugin /csi-diskplugin
ENTRYPOINT ["/csi-diskplugin"]