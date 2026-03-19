#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset
if [[ ${TRACE-} == "1" ]]; then
    set -o xtrace
fi

if [[ $# -eq 0 ]] || [[ $1 =~ -h|--help ]]; then
    prog=$(basename "$0")
    cat <<EOM
Wrapper around go tool pprof for dyntrace-operator components.

COMPONENTS:
  - operator
  - webhook
  - server
  - provisioner

ENDPOINTS:
  - allocs
  - block
  - goroutine
  - heap
  - mutex
  - profile
  - threadcreate

EXAMPLES:
    # Open interactive pprof shell
    $prog operator heap

    # Serve Web UI on port 8080
    $prog webhook profile?seconds=30 -http :8080

USAGE:
    $prog component endpoint [... pprof args]
EOM
    exit 0
fi

case "${1?missing component name}" in
    operator)
        RESOURCE=deployment/dynatrace-operator
        PORT=6060:6060
        ;;
    webhook)
        RESOURCE=deployment/dynatrace-webhook
        PORT=6060:6060
        ;;
    server)
        RESOURCE=daemonset/dynatrace-oneagent-csi-driver
        PORT=6060:6060
        ;;
    provisioner)
        RESOURCE=daemonset/dynatrace-oneagent-csi-driver
        PORT=6060:6061
        ;;
    *)
        echo "unknown component: $1" >&2
        exit 1
esac

ENDPOINT=${2?missing pprof endpoint}
ENDPOINT=${ENDPOINT#/debug*}
ENDPOINT=${ENDPOINT#/pprof*}
QUERY=${ENDPOINT#*\?}
if [[ $QUERY == "$ENDPOINT" ]]; then
    QUERY=
else
    QUERY="?${QUERY}"
    ENDPOINT=${ENDPOINT%\?*}
fi

case "$ENDPOINT" in
    allocs|block|goroutine|heap|mutex|threadcreate|profile) ;;
    *)
        echo "unsupported endpoint: $2" >&2
        exit 1
esac

kubectl port-forward -n dynatrace $RESOURCE $PORT &
CHILD_PID=$!
trap 'kill -TERM $CHILD_PID' EXIT

# Platform agnostic listen for socket. Do an initial wait to not spam the logs with errors
sleep 2
# shellcheck disable=SC2188
while ! </dev/tcp/localhost/6060; do
    sleep 2
done

# Consume two args to allow use of $@
shift
shift
go tool pprof "$@" "http://localhost:6060/debug/pprof/${ENDPOINT}${QUERY}"
