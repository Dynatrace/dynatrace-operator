# check=skip=RedundantTargetPlatform
# setup build image
FROM --platform=$BUILDPLATFORM golang:1.26.4@sha256:32c0e6e5c4f6707717051091b4d0b077464a679eaab563e11474efc5328e2aa5 AS operator-build

WORKDIR /app

ARG DEBUG_TOOLS
# renovate depName=github.com/go-delve/delve/cmd/dlv
RUN if [ "$DEBUG_TOOLS" = "true" ]; then GOBIN=/app/build/_output/bin go install github.com/go-delve/delve/cmd/dlv@v1.27.0; fi

# renovate depName=github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod
RUN go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.10.0

COPY go.mod go.sum ./
RUN go mod download -x

COPY pkg ./pkg
COPY cmd ./cmd
COPY .git ./.git

ARG GO_LINKER_ARGS
ARG GO_BUILD_TAGS
ARG TARGETARCH
ARG TARGETOS
ARG GOFIPS140=off

RUN --mount=type=cache,target="/root/.cache/go-build" \
    --mount=type=cache,target="/go/pkg" \
    CGO_ENABLED=0 GOFIPS140=$GOFIPS140 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -tags "${GO_BUILD_TAGS}" -trimpath -ldflags="${GO_LINKER_ARGS}" \
    -o ./build/_output/bin/dynatrace-operator ./cmd/

RUN cyclonedx-gomod app -licenses -assert-licenses -json -main cmd/ -output ./build/_output/bin/dynatrace-operator-bin-sbom.cdx.json

# platform is required, otherwise the copy command will copy the wrong architecture files, don't trust GitHub Actions linting warnings
FROM --platform=$TARGETPLATFORM registry.access.redhat.com/ubi9-micro:9.8-1779858820@sha256:b498b3ea26111ab4b81d65139f2ebd2ef9a2abb7a4588b7fdcc54889f95e9caa AS base
FROM --platform=$TARGETPLATFORM registry.access.redhat.com/ubi9:9.8-1780900431@sha256:46d19c10caf9888e8a01131283eaaf50c7f5d4eddab02cd92a66f8adf2e15407 AS dependency
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

# operator binary + sbom
COPY --from=operator-build /app/build/_output/bin /usr/local/bin

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

ARG GODEBUG_ARG

ENV OPERATOR=dynatrace-operator \
    USER_UID=1001 \
    USER_NAME=dynatrace-operator \
    GODEBUG=${GODEBUG_ARG:+fips140=only}

RUN /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}:${USER_UID}
