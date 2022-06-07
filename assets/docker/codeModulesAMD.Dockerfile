FROM alpine AS builder

ARG APIURL
ARG APITOKEN
ARG AGENTVERSION
ARG FLAVOR=multidistro
ARG ARCH=x86

RUN apk update && apk add --update jq
RUN mkdir data
RUN wget "${APIURL}/v1/deployment/installer/agent/unix/paas/version/${AGENTVERSION}/checksum?flavor=${FLAVOR}&arch=${ARCH}&bitness=all&skipMetadata=true" --header "Authorization: Api-Token ${APITOKEN}" -O checksum
RUN wget "${APIURL}/v1/deployment/installer/agent/unix/paas/version/${AGENTVERSION}?flavor=${FLAVOR}&arch=${ARCH}&bitness=all&skipMetadata=true" --header "Authorization: Api-Token ${APITOKEN}" -O /agent.zip
RUN [ "$(jq .sha256 -r checksum)" == "$(sha256sum agent.zip | awk '{ print $1 }')" ]
RUN unzip /agent.zip -d /data

FROM scratch
COPY --from=builder /data /
