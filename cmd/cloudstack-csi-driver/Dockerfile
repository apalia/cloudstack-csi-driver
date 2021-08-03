FROM alpine:3.14.0

LABEL \
    org.opencontainers.image.description="CloudStack CSI driver" \
    org.opencontainers.image.source="https://github.com/apalia/cloudstack-csi-driver/"

RUN apk add --no-cache \
    ca-certificates \
    # Provides mkfs.ext2, mkfs.ext3, mkfs.ext4 (used by k8s.io/mount-utils)
    e2fsprogs \
    # Provides mkfs.xfs
    xfsprogs \
    # Provides blkid, also used by k8s.io/mount-utils
    blkid \
    eudev \
    bash

COPY ./bin/cloudstack-csi-driver /cloudstack-csi-driver
COPY rescan-scsi-bus.sh /usr/bin/
RUN chmod +x /usr/bin/rescan-scsi-bus.sh
ENTRYPOINT ["/cloudstack-csi-driver"]