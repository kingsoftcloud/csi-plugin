FROM golang:1.11-alpine as builder
COPY . $GOPATH/src/csi-plugin/
WORKDIR $GOPATH/src/csi-plugin
RUN apk add make && make compile

FROM alpine
LABEL maintainers="ksc"

RUN apk -U upgrade && \
    apk -U add ca-certificates && \
    update-ca-certificates
COPY --from=builder  /go/src/csi-plugin/csi-diskplugin /csi-diskplugin
ENTRYPOINT ["/csi-diskplugin"]