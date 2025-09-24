# check=skip=RedundantTargetPlatform
# setup build image
FROM --platform=$BUILDPLATFORM golang:1.25.1@sha256:76a94c4a37aaab9b1b35802af597376b8588dc54cd198f8249633b4e117d9fcc AS operator-build

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
FROM --platform=$TARGETPLATFORM registry.access.redhat.com/ubi9-micro:9.6-1754467928@sha256:f5c5213d2969b7b11a6666fc4b849d56b48d9d7979b60a37bb853dff0255c14b AS base
FROM --platform=$TARGETPLATFORM registry.access.redhat.com/ubi9:9.6-1758184894@sha256:dbc1e98d14a022542e45b5f22e0206d3f86b5bdf237b58ee7170c9ddd1b3a283 AS dependency
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
COPY --from=registry.k8s.io/sig-storage/livenessprobe:v2.16.0@sha256:88092d100909918ae0a768956cf78c88bc59cd7232720f7cdbdfb5d2e235001e /livenessprobe /usr/local/bin

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
