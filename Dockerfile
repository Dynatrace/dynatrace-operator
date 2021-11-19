FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4 AS package-download

RUN microdnf install unzip

FROM golang:1.16-alpine AS operator-build

RUN apk update --no-cache && \
    apk add --no-cache gcc musl-dev btrfs-progs-dev lvm2-dev device-mapper-static && \
    rm -rf /var/cache/apk/*

ARG GO_BUILD_ARGS
COPY . /app
WORKDIR /app

RUN go get github.com/google/go-licenses && go-licenses save ./... --save_path third_party_licenses --force
RUN go get -d ./...

RUN CGO_ENABLED=1 go build "$GO_BUILD_ARGS" -o ./build/_output/bin/dynatrace-operator ./cmd/operator/

FROM registry.access.redhat.com/ubi8/ubi-micro:8.4

COPY --from=k8s.gcr.io/sig-storage/csi-node-driver-registrar:v2.3.0 /csi-node-driver-registrar /usr/local/bin
COPY --from=k8s.gcr.io/sig-storage/livenessprobe:v2.5.0 /livenessprobe /usr/local/bin

# copy tools required by init.sh
COPY --from=package-download /usr/bin/unzip /usr/local/bin/unzip
COPY --from=package-download /usr/bin/curl /usr/local/bin/curl

# copy curl dependencies
COPY --from=package-download /etc/pki/tls/certs /etc/pki/tls/certs
COPY --from=package-download /etc/pki/ca-trust/extracted /etc/pki/ca-trust/extracted
COPY --from=package-download /usr/lib64/libcurl.* \
     /usr/lib64/libssl.* \
     /usr/lib64/libcrypt* \
     /usr/lib64/libz.* \
     /usr/lib64/libnghttp2.* \
     /usr/lib64/libidn2.* \
     /usr/lib64/libssh.* \
     /usr/lib64/libpsl.* \
     /usr/lib64/libgssapi_krb5.* \
     /usr/lib64/libkrb5.* \
     /usr/lib64/libk5crypto.* \
     /usr/lib64/libcom_err.* \
     /usr/lib64/libldap-* \
     /usr/lib64/liblber-* \
     /usr/lib64/libbrotlidec* \
     /usr/lib64/libunistring.* \
     /usr/lib64/libkrb5support.* \
     /usr/lib64/libkeyutils.* \
     /usr/lib64/libsasl2.* \
     /usr/lib64/libbz2.* \
     /usr/lib64/libbrotlicommon.* /usr/lib64/

COPY --from=operator-build /app/build/_output/bin /usr/local/bin
COPY --from=operator-build /app/third_party_licenses /usr/share/dynatrace-operator/third_party_licenses

LABEL name="Dynatrace ActiveGate Operator" \
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
COPY build/bin /usr/local/bin

RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
