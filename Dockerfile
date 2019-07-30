FROM alpine
LABEL maintainers="ksc"

RUN apk -U upgrade && \
    apk add e2fsprogs && apk add blkid && apk add findmnt && \
    apk -U add ca-certificates && \
    update-ca-certificates
COPY bin/csi-diskplugin /csi-diskplugin
ENTRYPOINT ["/csi-diskplugin"]
