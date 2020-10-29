#!/bin/bash

set -eu

for arg in "$@"; do
  case $arg in
  --api-url)
    API_URL="$2"
    shift
    shift
    ;;
  --api-token)
    API_TOKEN="$2"
    shift
    shift
    ;;
  --paas-token)
    PAAS_TOKEN="$2"
    shift
    shift
    ;;
  --enable-k8s-monitoring)
    ENABLE_K8S_MONITORING="$2"
    shift
    shift
    ;;
  --set-app-log-content-access)
    SET_APP_LOG_CONTENT_ACCESS="$2"
    shift
    shift
    ;;
  --skip-cert-check)
    SKIP_CERT_CHECK="$2"
    shift
    shift
    ;;
  --enable-volume-storage)
    ENABLE_VOLUME_STORAGE="$2"
    shift
    shift
    ;;
  --connection-name)
    CONNECTION_NAME="$2"
    shift
    shift
    ;;
  esac
done

if [[ -z "$API_URL" ]]; then
  echo "Error: api-url not set!"
  exit 1
fi

if [[ -z "$API_TOKEN" ]]; then
  echo "Error: api-token not set!"
  exit 1
fi

if [[ -z "$PAAS_TOKEN" ]]; then
  echo "Error: paas-token not set!"
  exit 1
fi

if [[ "$ENABLE_K8S_MONITORING" ]]; then
  if [[ -z "$CONNECTION_NAME" ]]; then
    echo "Error: no name set for connection!"
    exit 1
  fi
fi

if [[ -z "${SET_APP_LOG_CONTENT_ACCESS:-}" ]]; then
  SET_APP_LOG_CONTENT_ACCESS=false
fi

if [[ -z "${SKIP_CERT_CHECK:-}" ]]; then
  SKIP_CERT_CHECK=false
fi

applyOneAgentOperator() {
  set +e
  if [[ ! $(oc get ns dynatrace &>/dev/null) ]]; then
    oc adm new-project --node-selector="" dynatrace
  fi

  if [[ ! $(oc get deployment dynatrace-oneagent-operator -n dynatrace &>/dev/null) ]]; then
    oc apply -f https://github.com/Dynatrace/dynatrace-oneagent-operator/releases/latest/download/openshift.yaml
  fi

  if [[ ! $(oc get secret oneagent -n dynatrace &>/dev/null) ]]; then
    oc -n dynatrace create secret generic oneagent --from-literal="apiToken=${API_TOKEN}" --from-literal="paasToken=${PAAS_TOKEN}"
  fi
  set -e
}

applyOneAgentCR() {
  # Apply OneAgent CR
  set +e
  if [[ ! $(oc get oneagent oneagent -n dynatrace &>/dev/null) ]]; then
    if [[ -z "${ENABLE_VOLUME_STORAGE:-}" ]]; then
      read -r -d '' oneagent <<EOF
apiVersion: dynatrace.com/v1alpha1
kind: OneAgent
metadata:
  name: oneagent
  namespace: dynatrace
spec:
  apiUrl: ${API_URL}
  tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
    operator: Exists
  skipCertCheck: ${SKIP_CERT_CHECK}
  args:
  - --set-app-log-content-access=${SET_APP_LOG_CONTENT_ACCESS}
  env:
  - name: ONEAGENT_ENABLE_VOLUME_STORAGE
    value: "true"
EOF
    else
      read -r -d '' oneagent <<EOF
apiVersion: dynatrace.com/v1alpha1
kind: OneAgent
metadata:
  name: oneagent
  namespace: dynatrace
spec:
  apiUrl: ${API_URL}
  tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
    operator: Exists
  skipCertCheck: ${SKIP_CERT_CHECK}
  args:
  - --set-app-log-content-access=${SET_APP_LOG_CONTENT_ACCESS}
EOF
    fi

    echo "$oneagent" | oc apply -f -
  fi
  set -e
}

applyDynatraceOperator() {
  set +e
  if [[ ! $(oc get deployment dynatrace-operator -n dynatrace &>/dev/null) ]]; then
    oc apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/openshift.yaml
  fi
  set -e
}

applyDynaKubeCR() {
  # Apply Dynakube CR
  set +e
  if [[ ! $(oc get dynakube dynakube -n dynatrace &>/dev/null) ]]; then
    cat <<EOF | kubectl apply -f -
apiVersion: dynatrace.com/v1alpha1
kind: DynaKube
metadata:
  name: dynakube
spec:
  apiUrl: ${API_URL}
  tokens: oneagent
  kubernetesMonitoring:
    enabled: true
    replicas: 1
EOF
  fi
  set -e
}

addK8sConfiguration() {
  # Set up K8s integration
  K8S_ENDPOINT="$(oc config view --minify -o jsonpath='{.clusters[0].cluster.server}')"
  if [[ -z "$K8S_ENDPOINT" ]]; then
    echo "Error: failed to get kubernetes endpoint!"
    exit 1
  fi

  K8S_SECRET_NAME="$(oc get sa dynatrace-kubernetes-monitoring -o jsonpath='{.secrets[1].name}' -n dynatrace)"
  if [[ -z "$K8S_SECRET_NAME" ]]; then
    echo "Error: failed to get kubernetes-monitoring secret!"
    exit 1
  fi

  K8S_BEARER="$(oc get secret "${K8S_SECRET_NAME}" -o jsonpath='{.data.token}' -n dynatrace | base64 --decode)"
  if [[ -z "$K8S_BEARER" ]]; then
    echo "Error: failed to get bearer token!"
    exit 1
  fi

  json=$(
    cat <<EOF
{
  "label": "${CONNECTION_NAME}",
  "endpointUrl": "${K8S_ENDPOINT}",
  "eventsFieldSelectors": [
    {
      "label": "Node events",
      "fieldSelector": "involvedObject.kind=Node",
      "active": true
    }
  ],
  "workloadIntegrationEnabled": false,
  "eventsIntegrationEnabled": true,
  "authToken": "${K8S_BEARER}",
  "active": true,
  "certificateCheckEnabled": true
}
EOF
  )

  curl -sS -X POST "${API_URL}/config/v1/kubernetes/credentials" \
    -H "accept: application/json; charset=utf-8" \
    -H "Authorization: Api-Token ${API_TOKEN}" \
    -H "Content-Type: application/json; charset=utf-8" \
    -d "${json}"
}

####### MAIN #######
applyOneAgentOperator
applyOneAgentCR

if [[ $ENABLE_K8S_MONITORING ]]; then
  applyDynatraceOperator
  applyDynaKubeCR
  addK8sConfiguration
fi
