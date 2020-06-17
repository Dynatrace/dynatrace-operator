FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

LABEL name="Dynatrace ActiveGate Operator" \
      vendor="Dynatrace LLC" \
      maintainer="Dynatrace LLC" \
      version="1.x" \
      release="1" \
      url="https://www.dynatrace.com"
      # summary="Dynatrace is an all-in-one, zero-config monitoring platform designed by and for cloud natives. It is powered by artificial intelligence that identifies performance problems and pinpoints their root causes in seconds." \
      # description="Dynatrace OneAgent automatically discovers all technologies, services and applications that run on your host."

ENV OPERATOR=dynatrace-activegate-operator \
    USER_UID=1001 \
    USER_NAME=dynatrace-activegate-operator

RUN  microdnf install unzip && microdnf clean all
COPY LICENSE /licenses/
COPY build/_output/bin /usr/local/bin
COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
