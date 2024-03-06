# setup build image
FROM golang:1.21.6@sha256:7026fb72cfa9cc112e4d1bf4b35a15cac61a413d0252d06615808e7c987b33a7 AS go-base
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
ARG GO_BUILD_TAGS
COPY pkg ./pkg
COPY cmd ./cmd
RUN CGO_ENABLED=1 CGO_CFLAGS="-O2 -Wno-return-local-addr" \
    go build -tags "${GO_BUILD_TAGS}" -trimpath -ldflags="${GO_LINKER_ARGS}" \
    -o ./build/_output/bin/dynatrace-operator ./cmd/

FROM registry.access.redhat.com/ubi9-micro:9.3-15@sha256:8e33df2832f039b4b1adc53efd783f9404449994b46ae321ee4a0bf4499d5c42 AS base
FROM registry.access.redhat.com/ubi9:9.3-1610@sha256:66233eebd72bb5baa25190d4f55e1dc3fff3a9b77186c1f91a0abdb274452072 AS dependency
RUN mkdir -p /tmp/rootfs-dependency
COPY --from=base / /tmp/rootfs-dependency
RUN dnf install --installroot /tmp/rootfs-dependency \
      util-linux-core tar \
      --releasever 9 \
      --setopt install_weak_deps=false \
      --nodocs -y \
 && dnf --installroot /tmp/rootfs-dependency clean all \
 && rm -rf \
      /tmp/rootfs-dependency/var/cache/* \
      /tmp/rootfs-dependency/var/log/dnf* \
      /tmp/rootfs-dependency/var/log/yum.*

FROM base

COPY --from=dependency /tmp/rootfs-dependency /

# operator binary
COPY --from=operator-build /app/build/_output/bin /usr/local/bin

# csi binaries
COPY --from=registry.k8s.io/sig-storage/csi-node-driver-registrar:v2.9.3@sha256:0f64602ea791246712b51df334bbd701a0f31df9950a4cb9c28c059f367baa9e /csi-node-driver-registrar /usr/local/bin
COPY --from=registry.k8s.io/sig-storage/livenessprobe:v2.11.0@sha256:82adbebdf5d5a1f40f246aef8ddbee7f89dea190652aefe83336008e69f9a89f /livenessprobe /usr/local/bin

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
      summary="The Dynatrace Operator is an open source Kubernetes Operator for easily deploying and managing Dynatrace components for Kubernetes / OpenShift observability. By leveraging the Dynatrace Operator you can innovate faster with the full potential of Kubernetes / OpenShift and Dynatrace’s best-in-class observability and intelligent automation." \
      description="Automate Kubernetes observability with Dynatrace" \
      io.k8s.description="Automate Kubernetes observability with Dynatrace" \
      io.k8s.display-name="Dynatrace Operator" \
      io.openshift.tags="observability,monitoring,dynatrace,operator,logging,metrics,tracing,prometheus,alerts" \
      vcs-url="https://github.com/Dynatrace/dynatrace-operator.git" \
      vcs-type="git" \
      changelog-url="https://github.com/Dynatrace/dynatrace-operator/releases"

ENV OPERATOR=dynatrace-operator \
    USER_UID=1001 \
    USER_NAME=dynatrace-operator

RUN /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}:${USER_UID}
