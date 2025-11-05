# check=skip=RedundantTargetPlatform
# setup build image
FROM --platform=$BUILDPLATFORM golang:1.25.3@sha256:2e3aca25948111e2a3e4acd66b5c5abebecdea6a434eead036152631e4d0b3a0 AS operator-build

WORKDIR /app

ARG DEBUG_TOOLS
RUN if [ "$DEBUG_TOOLS" = "true" ]; then \
      GOBIN=/app/build/_output/bin go install github.com/go-delve/delve/cmd/dlv@latest; \
    fi

COPY go.mod go.sum ./
RUN go mod download -x

COPY pkg ./pkg
COPY cmd ./cmd
COPY .git /.git

ARG GO_LINKER_ARGS
ARG GO_BUILD_TAGS
ARG TARGETARCH
ARG TARGETOS


RUN --mount=type=cache,target="/root/.cache/go-build" \
    --mount=type=cache,target="/go/pkg" \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -tags "${GO_BUILD_TAGS}" -trimpath -ldflags="${GO_LINKER_ARGS}" \
    -o ./build/_output/bin/dynatrace-operator ./cmd/

# renovate depName=github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod
RUN go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.9.0
RUN cyclonedx-gomod app -licenses -assert-licenses -json -main cmd/ -output ./build/_output/bin/dynatrace-operator-bin-sbom.cdx.json

# platform is required, otherwise the copy command will copy the wrong architecture files, don't trust GitHub Actions linting warnings
FROM --platform=$TARGETPLATFORM registry.access.redhat.com/ubi9-micro:9.6-1762316544@sha256:990002083442f6a93cd3249da32ecb7c3f6be778a1bec3a73a9c17fbc40edc15 AS base
FROM --platform=$TARGETPLATFORM registry.access.redhat.com/ubi9:9.6-1760340943@sha256:dec374e05cc13ebbc0975c9f521f3db6942d27f8ccdf06b180160490eef8bdbc AS dependency
RUN mkdir -p /tmp/rootfs-dependency
COPY --from=base / /tmp/rootfs-dependency
RUN dnf install --installroot /tmp/rootfs-dependency \
      util-linux-core \
      --releasever 9 \
      --setopt install_weak_deps=false \
      --nodocs -y \
 && dnf --installroot /tmp/rootfs-dependency clean all \
 && rm -rf \
      /tmp/rootfs-dependency/var/cache/* \
      /tmp/rootfs-dependency/var/log/dnf* \
      /tmp/rootfs-dependency/var/log/yum.*

# platform is required, otherwise the copy command will copy the wrong architecture files, don't trust GitHub Actions linting warnings
FROM --platform=$TARGETPLATFORM base

COPY --from=dependency /tmp/rootfs-dependency /

# operator binary
COPY --from=operator-build /app/build/_output/bin /usr/local/bin

# csi binaries
COPY --from=registry.k8s.io/sig-storage/csi-node-driver-registrar:v2.15.0@sha256:11f199f6bec47403b03cb49c79a41f445884b213b382582a60710b8c6fdc316a /csi-node-driver-registrar /usr/local/bin
COPY --from=registry.k8s.io/sig-storage/livenessprobe:v2.17.0@sha256:9b75b9ade162136291d5e8f13a1dfc3dec71ee61419b1bfc112e0796ff8a6aa9 /livenessprobe /usr/local/bin

COPY ./third_party_licenses /usr/share/dynatrace-operator/third_party_licenses
COPY LICENSE /licenses/

# operator sbom
COPY --from=operator-build ./app/build/_output/bin/dynatrace-operator-bin-sbom.cdx.json ./dynatrace-operator-bin-sbom.cdx.json

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
