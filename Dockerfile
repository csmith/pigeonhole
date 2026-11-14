FROM golang:1.27.4 AS build
WORKDIR /go/src/app
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    set -eux; \
    CGO_ENABLED=0 GO111MODULE=on go install .; \
    go tool go-licenses save ./... --save_path=/notices \
    ; \
    mkdir -p /mounts/data;

FROM ghcr.io/greboid/dockerbase/nonroot:1.20261007.0
COPY --from=build /go/bin/pigeonhole /pigeonhole
COPY --from=build /notices /notices
COPY --from=build --chown=65532:65532 /mounts /
VOLUME /data
ENV DB=/data/pigeonhole.db
ENTRYPOINT ["/pigeonhole"]
