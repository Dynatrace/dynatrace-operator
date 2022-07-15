FROM golang:1.18-alpine AS operator-build

RUN apk update --no-cache && \
    apk add --no-cache gcc musl-dev btrfs-progs-dev lvm2-dev device-mapper-static gpgme-dev git && \
    rm -rf /var/cache/apk/*

ARG GO_LINKER_ARGS
COPY . /app
WORKDIR /app

# move previously cached go modules to gopath
RUN if [ -d ./mod ]; then mkdir -p ${GOPATH}/pkg && [ -d mod ] && mv ./mod ${GOPATH}/pkg; fi;

RUN CGO_ENABLED=1 go build -ldflags="${GO_LINKER_ARGS}" -o ./build/_output/bin/dynatrace-operator ./src/cmd/

FROM registry.access.redhat.com/ubi8-minimal:8.6 as dependency-src

RUN  microdnf install util-linux && microdnf clean all

FROM registry.access.redhat.com/ubi8-micro:8.6

# operator dependencies
COPY --from=operator-build /etc/ssl/cert.pem /etc/ssl/cert.pem
COPY --from=operator-build /app/build/_output/bin /usr/local/bin

COPY --from=operator-build /lib/libc.musl-*.so.* /lib/
COPY --from=operator-build /lib/ld-musl-*.so.* /lib/

COPY --from=operator-build /lib/libdevmapper.so.* /lib/

COPY --from=operator-build /usr/lib/libassuan.so.* /usr/lib/
COPY --from=operator-build /usr/lib/libgpg-error.so.* /usr/lib/
COPY --from=operator-build /usr/lib/libgpgme.so.* /usr/lib/

# csi binaries
COPY --from=k8s.gcr.io/sig-storage/csi-node-driver-registrar:v2.5.1 /csi-node-driver-registrar /usr/local/bin
COPY --from=k8s.gcr.io/sig-storage/livenessprobe:v2.7.0 /livenessprobe /usr/local/bin

# csi depdenencies
COPY --from=dependency-src /bin/mount /bin/umount /bin/
COPY --from=dependency-src /lib64/libmount.so.1 /lib64/libblkid.so.1 /lib64/libuuid.so.1 /lib64/

COPY ./third_party_licenses /usr/share/dynatrace-operator/third_party_licenses

LABEL name="Dynatrace Operator" \
      vendor="Dynatrace LLC" \
      maintainer="Dynatrace LLC" \
      version="1.x" \
      release="1" \
      url="https://www.dynatrace.com" \
      summary="Dynatrace is an all-in-one, zero-config monitoring platform designed by and for cloud natives. It is powered by artificial intelligence that identifies performance problems and pinpoints their root causes in seconds." \
      description="ActiveGate works as a proxy between Dynatrace OneAgent and Dynatrace Cluster"

ENV OPERATOR=dynatrace-operator \
    USER_UID=1001 \
    USER_NAME=dynatrace-operator

COPY LICENSE /licenses/
COPY hack/build/bin /usr/local/bin

RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}:${USER_UID}
