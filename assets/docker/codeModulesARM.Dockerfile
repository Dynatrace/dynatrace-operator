FROM alpine@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS builder

ARG APIURL
ARG APITOKEN
ARG AGENTVERSION
ARG FLAVOR=default
ARG ARCH=arm

RUN apk update && apk add --update jq
RUN mkdir data
RUN wget "${APIURL}/v1/deployment/installer/agent/unix/paas/version/${AGENTVERSION}/checksum?flavor=${FLAVOR}&arch=${ARCH}&bitness=all&skipMetadata=true" --header "Authorization: Api-Token ${APITOKEN}" -O checksum
RUN wget "${APIURL}/v1/deployment/installer/agent/unix/paas/version/${AGENTVERSION}?flavor=${FLAVOR}&arch=${ARCH}&bitness=all&skipMetadata=true" --header "Authorization: Api-Token ${APITOKEN}" -O /agent.zip
RUN [ "$(jq .sha256 -r checksum)" == "$(sha256sum agent.zip | awk '{ print $1 }')" ]
RUN unzip /agent.zip -d /data

FROM scratch
COPY --from=builder /data /opt/dynatrace/oneagent
