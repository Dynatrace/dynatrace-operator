FROM mcr.microsoft.com/oss/go/microsoft/golang:1.25.0-fips-bookworm@sha256:a447152dc3f2825e475305f91047e1fd3e2510565394e7f17e0c245e257b9dc5 AS operator-build

ENV GOEXPERIMENT=systemcrypto

WORKDIR /app

ARG DEBUG_TOOLS
# renovate depName=github.com/go-delve/delve/cmd/dlv
RUN if [ "$DEBUG_TOOLS" = "true" ]; then GOBIN=/app/build/_output/bin go install github.com/go-delve/delve/cmd/dlv@v1.26.0; fi

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
    CGO_ENABLED=1 GOFIPS=1 \
    go build -tags "${GO_BUILD_TAGS}" -trimpath -ldflags="${GO_LINKER_ARGS}" \
    -o ./build/_output/bin/dynatrace-operator ./cmd/

# renovate depName=github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod
RUN go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.10.0
RUN CGO_ENABLED=1 cyclonedx-gomod app -licenses -assert-licenses -json -main cmd/ -output ./build/_output/bin/dynatrace-operator-bin-sbom.cdx.json

# platform is required, otherwise the copy command will copy the wrong architecture files, don't trust GitHub Actions linting warnings
FROM registry.access.redhat.com/ubi9-micro:9.7-1771346390@sha256:093a704be0eaef9bb52d9bc0219c67ee9db13c2e797da400ddb5d5ae6849fa10 AS base
FROM registry.access.redhat.com/ubi9:9.7-1770238273@sha256:b8923f58ef6aebe2b8f543f8f6c5af15c6f9aeeef34ba332f33bf7610012de0c AS dependency
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

# install openssl-dependencies
RUN dnf install --setopt install_weak_deps=false --nodocs -y make gcc perl

WORKDIR /openssl_build
# build and install openssl
# version must be FIPS certified, details https://openssl-library.org/source/
# get the sha256 from trusted source (e.g. Github release, https://github.com/openssl/openssl/releases)
ENV OPENSSL_BUILD_VERSION="3.1.2"
ENV OPENSSL_BUILD_TARBALL_SHA256="a0ce69b8b97ea6a35b96875235aa453b966ba3cba8af2de23657d8b6767d6539"
ENV OPENSSL_BUILD_CONFIGURE_ARGS="enable-fips"

RUN curl -L -o openssl.tar.gz https://github.com/openssl/openssl/releases/download/openssl-${OPENSSL_BUILD_VERSION}/openssl-${OPENSSL_BUILD_VERSION}.tar.gz && \
    sha256sum --quiet -c - <<< "${OPENSSL_BUILD_TARBALL_SHA256}  openssl.tar.gz" && \
    tar --strip-components=1 -xzf openssl.tar.gz

# disable the aflag test because it doesn't work on qemu (aka cross compile, see https://github.com/openssl/openssl/pull/17945)
RUN if [ "$TARGETPLATFORM" = "linux/amd64" ]; then \
        ./Configure ${OPENSSL_BUILD_CONFIGURE_ARGS} && make && make test TESTS="-test_afalg"; \
    else \
        echo "skipping -test_afalg"; \
        ./Configure ${OPENSSL_BUILD_CONFIGURE_ARGS} && make; \
    fi

# do not install man pages
RUN make DESTDIR=/tmp/rootfs-dependency install_sw install_ssldirs install_fips

# platform is required, otherwise the copy command will copy the wrong architecture files, don't trust GitHub Actions linting warnings
FROM base

ARG TARGETPLATFORM

COPY --from=dependency /tmp/rootfs-dependency /

# operator binary
COPY --from=operator-build /app/build/_output/bin /usr/local/bin

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

# FIPS
ENV LIBGCRYPT_FORCE_FIPS_MODE=1
ENV GOFIPS=1

# generate openssl FIPS config and run self-tests (these are required to be compliant)
RUN case "${TARGETPLATFORM}" in \
        *amd64) LIB_DIR=/usr/local/lib64 ;; \
        *arm64) LIB_DIR=/usr/local/lib ;; \
        *) echo $TARGETPLATFORM ; exit 2 ;; \
    esac; \
    # Otherwise openssl will still use system libs
    ldconfig "${LIB_DIR}" && \
    openssl fipsinstall -out /usr/local/ssl/fipsmodule.cnf -module "${LIB_DIR}/ossl-modules/fips.so"

# always use FIPS (sets the default openssl config to use the FIPS provider), also the config dir for the self-built openssl is /usr/local/ssl and NOT /etc/ssl
RUN sed -i '/\.include fipsmodule\.cnf/s/^# //g' /usr/local/ssl/openssl.cnf && sed -i '/fips = fips_sect/s/^# //g' /usr/local/ssl/openssl.cnf && sed -i 's#\.include fipsmodule\.cnf#\.include /usr/local/ssl/fipsmodule\.cnf#g' /usr/local/ssl/openssl.cnf

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}:${USER_UID}
