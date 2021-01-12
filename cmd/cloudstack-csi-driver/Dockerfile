FROM alpine:3.12.3

LABEL \
    org.opencontainers.image.description="CloudStack CSI driver" \
    org.opencontainers.image.source="https://github.com/apalia/cloudstack-csi-driver/"

RUN apk add --no-cache xfsprogs e2fsprogs blkid ca-certificates

COPY ./bin/cloudstack-csi-driver /cloudstack-csi-driver
ENTRYPOINT ["/cloudstack-csi-driver"]