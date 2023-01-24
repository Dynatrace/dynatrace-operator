# setup build image
FROM golang:1.19.5 AS go-base
RUN \
    --mount=type=cache,target=/var/cache/apt \
    apt-get update && apt-get install -y libbtrfs-dev libdevmapper-dev

# download go dependencies
FROM go-base AS go-mod
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# build operator binary
FROM go-mod AS operator-build
ARG GO_LINKER_ARGS
COPY src ./src
RUN CGO_ENABLED=1 CGO_CFLAGS="-O2 -Wno-return-local-addr" \
    go build -tags "containers_image_openpgp,osusergo,netgo,sqlite_omit_load_extension,containers_image_storage_stub,containers_image_docker_daemon_stub" -trimpath -ldflags="${GO_LINKER_ARGS}" \
    -o ./build/_output/bin/dynatrace-operator ./src/cmd/

# get extra dependencies
FROM registry.access.redhat.com/ubi9-minimal:9.1.0 as dependency-src
RUN \
    --mount=type=cache,target=/var/cache/dnf \
    microdnf install -y util-linux tar --nodocs

FROM registry.access.redhat.com/ubi9-micro:9.1.0

# operator binary
COPY --from=operator-build /app/build/_output/bin /usr/local/bin

# # trusted certificates
COPY --from=dependency-src /etc/ssl/cert.pem /etc/ssl/cert.pem

# csi dependencies
COPY --from=dependency-src /bin/mount /bin/umount /bin/tar /bin/
COPY --from=dependency-src /lib64/libmount.so.1 /lib64/libblkid.so.1 /lib64/libuuid.so.1 /lib64/

# csi binaries
COPY --from=registry.k8s.io/sig-storage/csi-node-driver-registrar:v2.7.0 /csi-node-driver-registrar /usr/local/bin
COPY --from=registry.k8s.io/sig-storage/livenessprobe:v2.9.0 /livenessprobe /usr/local/bin

COPY ./third_party_licenses /usr/share/dynatrace-operator/third_party_licenses
COPY LICENSE /licenses/

# custom scripts
COPY hack/build/bin /usr/local/bin

LABEL name="Dynatrace Operator" \
      vendor="Dynatrace LLC" \
      maintainer="Dynatrace LLC" \
      version="1.x" \
      release="1" \
      url="https://www.dynatrace.com" \
      summary="Dynatrace is an all-in-one, zero-config monitoring platform designed by and for cloud natives. It is powered by artificial intelligence that identifies performance problems and pinpoints their root causes in seconds." \
      description="ActiveGate works as a proxy between Dynatrace OneAgent and Dynatrace Cluster" \
      io.k8s.description="Dynatrace Operator image." \
      io.k8s.display-name="Dynatrace Operator" \
      io.openshift.tags="dynatrace-operator" \
      vcs-url="https://github.com/Dynatrace/dynatrace-operator.git" \
      vcs-type="git" \
      changelog-url="https://github.com/Dynatrace/dynatrace-operator/releases"

ENV OPERATOR=dynatrace-operator \
    USER_UID=1001 \
    USER_NAME=dynatrace-operator

RUN /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}:${USER_UID}
