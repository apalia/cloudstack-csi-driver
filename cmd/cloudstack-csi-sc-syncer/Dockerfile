FROM alpine:3.14.0

LABEL \
    org.opencontainers.image.description="CloudStack disk offering to Kubernetes storage class syncer" \
    org.opencontainers.image.source="https://github.com/apalia/cloudstack-csi-driver/"

RUN apk add --no-cache ca-certificates

COPY ./bin/cloudstack-csi-sc-syncer /cloudstack-csi-sc-syncer
ENTRYPOINT ["/cloudstack-csi-sc-syncer"]