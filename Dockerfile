FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

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

RUN  microdnf install unzip util-linux && microdnf clean all
COPY LICENSE /licenses/
COPY third_party_licenses /usr/share/dynatrace-operator/third_party_licenses
COPY build/_output/bin /usr/local/bin
COPY build/bin /usr/local/bin

COPY --from=k8s.gcr.io/sig-storage/csi-node-driver-registrar:v2.2.0 /csi-node-driver-registrar /usr/local/bin
COPY --from=k8s.gcr.io/sig-storage/livenessprobe:v2.3.0 /livenessprobe /usr/local/bin

RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
