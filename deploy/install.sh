#!/bin/sh

set -e

CLI="kubectl"
SKIP_CERT_CHECK="false"
ENABLE_VOLUME_STORAGE="false"
CLUSTER_NAME_REGEX="^[-_a-zA-Z0-9][-_\.a-zA-Z0-9]*$"
CLUSTER_NAME_LENGTH=256

for arg in "$@"; do
  case $arg in
  --api-url)
    API_URL="$2"
    shift 2
    ;;
  --api-token)
    API_TOKEN="$2"
    shift 2
    ;;
  --paas-token)
    PAAS_TOKEN="$2"
    shift 2
    ;;
  --skip-cert-check)
    SKIP_CERT_CHECK="true"
    shift
    ;;
  --enable-volume-storage)
    ENABLE_VOLUME_STORAGE="true"
    shift
    ;;
  --cluster-name)
    CLUSTER_NAME="$2"
    shift 2
    ;;
  --openshift)
    CLI="oc"
    shift
    ;;
  esac
done

if [ -z "$API_URL" ]; then
  echo "Error: api-url not set!"
  exit 1
fi

if [ -z "$API_TOKEN" ]; then
  echo "Error: api-token not set!"
  exit 1
fi

if [ -z "$PAAS_TOKEN" ]; then
  echo "Error: paas-token not set!"
  exit 1
fi

if [ -z "$CLUSTER_NAME" ]; then
  if ! echo "$CLUSTER_NAME" | grep -Eq "$CLUSTER_NAME_REGEX"; then
    echo "Error: cluster name does not match regex!"
    exit 1
  fi

  if [ "${#CLUSTER_NAME}" -ge $CLUSTER_NAME_LENGTH ]; then
    echo "Error: cluster name too long!"
    exit 1
  fi
fi

set -u

checkIfNSExists() {
  if ! "${CLI}" get ns dynatrace >/dev/null 2>&1; then
    if [ "${CLI}" = "kubectl" ]; then
      "${CLI}" create namespace dynatrace
    else
      "${CLI}" adm new-project --node-selector="" dynatrace
    fi
  else
    echo "Namespace already exists"
  fi
}

applyDynatraceOperator() {
  if [ "${CLI}" = "kubectl" ]; then
    "${CLI}" apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes.yaml
  else
    "${CLI}" apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/openshift.yaml
  fi

  "${CLI}" -n dynatrace create secret generic dynakube --from-literal="apiToken=${API_TOKEN}" --from-literal="paasToken=${PAAS_TOKEN}" --dry-run -o yaml | "${CLI}" apply -f -
}

applyDynaKubeCR() {
  cat <<EOF | "${CLI}" apply -f -
apiVersion: dynatrace.com/v1alpha1
kind: DynaKube
metadata:
  name: dynakube
  namespace: dynatrace
spec:
  apiUrl: ${API_URL}
  skipCertCheck: ${SKIP_CERT_CHECK}
  networkZone: ${CLUSTER_NAME}
  kubernetesMonitoring:
    enabled: true
    group: ${CLUSTER_NAME}
  routing:
    enabled: true
    group: ${CLUSTER_NAME}
  classicFullStack:
    enabled: true
    tolerations:
    - effect: NoSchedule
      key: node-role.kubernetes.io/master
      operator: Exists
    env:
    - name: ONEAGENT_ENABLE_VOLUME_STORAGE
      value: "${ENABLE_VOLUME_STORAGE}"
    args:
    - --set-host-group="${CLUSTER_NAME}"
EOF
}

addK8sConfiguration() {
  K8S_ENDPOINT="$("${CLI}" config view --minify -o jsonpath='{.clusters[0].cluster.server}')"
  if [ -z "$K8S_ENDPOINT" ]; then
    echo "Error: failed to get kubernetes endpoint!"
    exit 1
  fi

  K8S_SECRET_NAME="$(for token in $("${CLI}" get sa dynatrace-kubernetes-monitoring -o jsonpath='{.secrets[*].name}' -n dynatrace); do echo "$token"; done | grep token)"
  if [ -z "$K8S_SECRET_NAME" ]; then
    echo "Error: failed to get kubernetes-monitoring secret!"
    exit 1
  fi

  K8S_BEARER="$("${CLI}" get secret "${K8S_SECRET_NAME}" -o jsonpath='{.data.token}' -n dynatrace | base64 --decode)"
  if [ -z "$K8S_BEARER" ]; then
    echo "Error: failed to get bearer token!"
    exit 1
  fi

  json="$(
    cat <<EOF
{
  "label": "${CLUSTER_NAME}",
  "endpointUrl": "${K8S_ENDPOINT}",
  "eventsFieldSelectors": [
    {
      "label": "Node events",
      "fieldSelector": "involvedObject.kind=Node",
      "active": true
    }
  ],
  "workloadIntegrationEnabled": true,
  "eventsIntegrationEnabled": false,
  "authToken": "${K8S_BEARER}",
  "active": true,
  "certificateCheckEnabled": "${SKIP_CERT_CHECK}"
}
EOF
  )"

  response=$(apiRequest "POST" "/config/v1/kubernetes/credentials" "${json}")

  if echo "$response" | grep "${CLUSTER_NAME}" >/dev/null 2>&1; then
    echo "Kubernetes monitoring successfully setup."
  else
    echo "Error adding Kubernetes cluster to Dynatrace: $response"
  fi
}

checkForExistingCluster() {
  response=$(apiRequest "GET" "/config/v1/kubernetes/credentials" "")
  if echo "$response" | grep -FEq "\"name\":\"${CLUSTER_NAME}\""; then
    echo "Error: Cluster name already exists!"
    exit 1
  fi
}

checkTokenScopes() {
  jsonAPI="{\"token\": \"${API_TOKEN}\"}"
  jsonPaaS="{\"token\": \"${PAAS_TOKEN}\"}"

  responseAPI=$(apiRequest "POST" "/v1/tokens/lookup" "${jsonAPI}")

  if echo "$responseAPI" | grep -q "Authentication failed"; then
    echo "Error: API token authentication failed!"
    exit 1
  fi

  if ! echo "$responseAPI" | grep -q "WriteConfig"; then
    echo "Error: API token does not have config write permission!"
    exit 1
  fi

  if ! echo "$responseAPI" | grep -q "ReadConfig"; then
    echo "Error: API token does not have config read permission!"
    exit 1
  fi

  responsePaaS=$(apiRequest "POST" "/v1/tokens/lookup" "${jsonPaaS}")

  if echo "$responsePaaS" | grep -q "Token does not exist"; then
    echo "Error: PaaS token does not exist!"
    exit 1
  fi
}

apiRequest() {
  method=$1
  url=$2
  json=$3

  response="$(curl -sS -X ${method} "${API_URL}${url}" \
    -H "accept: application/json; charset=utf-8" \
    -H "Authorization: Api-Token ${API_TOKEN}" \
    -H "Content-Type: application/json; charset=utf-8" \
    -d "${json}")"

  echo "$response"
}

####### MAIN #######
printf "\nCheck for token scopes...\n"
checkTokenScopes
printf "\nCheck if cluster already exists...\n"
checkForExistingCluster
printf "\nCreating Dynatrace namespace...\n"
checkIfNSExists
printf "\nApplying Dynatrace Operator...\n"
applyDynatraceOperator
printf "\nApplying DynaKube CustomResource...\n"
applyDynaKubeCR
printf "\nAdding cluster to Dynatrace...\n"
addK8sConfiguration
