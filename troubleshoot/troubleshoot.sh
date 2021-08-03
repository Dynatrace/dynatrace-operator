#!/bin/bash

set -e

cli="kubectl"
default_oneagent_image="docker.io/dynatrace/oneagent"

missing_value="<no value>"
selected_dynakube=""
selected_namespace="dynatrace"
api_url=""
paas_token=""

while [ $# -gt 0 ]; do
  case "$1" in
  --dynakube)
    selected_dynakube="$2"
    shift 2
    ;;
  --namespace)
    selected_namespace="$2"
    shift 2
    ;;
  --oc)
    cli="oc"
    shift 1
    ;;
  --openshift)
    default_oneagent_image="registry.connect.redhat.com/dynatrace/oneagent"
    shift 1
    ;;
  *)
    echo "Warning: skipping unsupported option: $1"
    shift
    ;;
  esac
done

log_info() {
  printf "[%s] %s\n" "$1" "$2"
}

error() {
  printf "ERROR: %s\n" "$1"
  exit 1
}

checkNs() {
  log_info "namespace" "checking if namespace '$selected_namespace' exists .."
  if ! "${cli}" get ns "$selected_namespace" >/dev/null 2>&1; then
    error "missing namespace '$selected_namespace'"
  else
    log_info "namespace" "using namespace '$selected_namespace'"
  fi
}

checkDynakube() {
  log_info "dynakube" "checking if Dynakube is configured correctly ..."

  # check dynakube crd exists
  crd="$("${cli}" get dynakube -n "$selected_namespace" >/dev/null 2>&1)"
  if [ -z "$crd" ]; then
    log_info "dynakube" "CRD for Dynakube exists"
  else
    error "CRD for Dynakube missing"
  fi

  # check dynakube cr exists
  if [[ -n "$selected_dynakube" ]]; then
    # dynakube set via parameter
    if ! "${cli}" get dynakube "${selected_dynakube}" -n "$selected_namespace" >/dev/null 2>&1; then
      error "Selected Dynakube '${selected_dynakube}' does not exist"
    fi
  else
    # dynakube not set, check for existing
    names="$("${cli}" get dynakube -n "$selected_namespace" -o jsonpath={..metadata.name})"
    if [ -z "$names" ]; then
      error "No Dynakube exists"
    fi

    read -ra names_arr <<<"$names"
    log_info "dynakube" "selecting Dynakube from available: '${names_arr[*]}'"
    selected_dynakube="${names_arr[0]}"
  fi

  log_info "dynakube" "'${selected_dynakube}' selected"

  checkApiUrl
  checkSecret

  log_info "dynakube" "'${selected_dynakube}' is valid"
}

checkApiUrl() {
  log_info "dynakube" "checking if api url is valid..."

  api_url=$("${cli}" get dynakube "${selected_dynakube}" -n "$selected_namespace" --template="{{.spec.apiUrl}}")
  if [ "${api_url##*/}" != "api" ]; then
    error "api url has to end on '/api'"
  fi
  # todo: check for valid url?

  log_info "dynakube" "api url is valid"
}

checkSecret() {
  log_info "dynakube" "checking if secret is valid ..."

  # use dynakube name or tokens value if set
  secret_name="$selected_dynakube"
  tokens=$("${cli}" get dynakube "${selected_dynakube}" -n "$selected_namespace" --template="{{.spec.tokens}}")
  if [[ "$tokens" != "$missing_value" ]]; then
    secret_name=$tokens
  fi

  if ! "${cli}" get secret "$secret_name" -n "$selected_namespace" &>/dev/null; then
    error "secret with the name '${secret_name}' is missing"
  else
    log_info "dynakube" "secret '$secret_name' exists"
  fi

  token_names=("apiToken" "paasToken")
  for token_name in "${token_names[@]}"; do
    token=$("${cli}" get secret "$secret_name" -n "$selected_namespace" --template="{{.data.${token_name}}}")
    if [[ "$token" == "$missing_value" ]]; then
      error "token '${token_name}' does not exist in secret '${secret_name}'"
    else
      log_info "dynakube" "secret token '$token_name' exists"
    fi

    if [ "$token_name" = "paasToken" ]; then
      paas_token=$(echo "$token" | base64 -d)
    fi
  done
}

getImage() {
  type="$1"

  if [[ "$type" == "oneAgent" ]] ; then
    # oneagent uses docker.io by default
    image="$default_oneagent_image"
  else
    # activegate is not published and uses the cluster registry by default
    api_url=$("${cli}" get dynakube "${selected_dynakube}" -n "$selected_namespace" --template="{{.spec.apiUrl}}")
    image="${api_url#*//}"
    image="${image%/*}/linux/activegate"
  fi

  dynakube_image=$("${cli}" get dynakube "${selected_dynakube}" -n "$selected_namespace" --template="{{.spec.${type}.image}}")
  if [[ -n "$dynakube_image" && "$dynakube_image" != "$missing_value" ]]; then
    image="$dynakube_image"
  fi

  echo "$image"
}

checkImagePullable() {
  container_cli="$1"

  log_info "image" "checking if image is pullable ..."

  # load pull secret
  custom_pull_secret_name=$("${cli}" get dynakube "${selected_dynakube}" -n "$selected_namespace" --template="{{.spec.customPullSecret}}")
  if [[ -n "$custom_pull_secret_name" && "$custom_pull_secret_name" != "$missing_value" ]] ; then
    pull_secret_name="$custom_pull_secret_name"
  else
    pull_secret_name="$selected_dynakube-pull-secret"
  fi
  log_info "image" "using pull secret '$pull_secret_name'"

  pull_secret_encoded=$("${cli}" get secret "$pull_secret_name" -n "$selected_namespace" -o "jsonpath={.data['\.dockerconfigjson']}")
  pull_secret="$(echo "$pull_secret_encoded" | base64 -d)"

  # load used images (default or custom)
  dynakube_oneagent_image=$(getImage "oneAgent")
  dynakube_activegate_image=$(getImage "activeGate")

  # split images into registry and image name
  oneagent_registry="${dynakube_oneagent_image%%/*}"
  oneagent_image="${dynakube_oneagent_image##"$oneagent_registry/"}"
  log_info "image" "using '$oneagent_image' on '$oneagent_registry' as oneagent image"

  activegate_registry="${dynakube_activegate_image%%/*}"
  activegate_image="${dynakube_activegate_image##"$activegate_registry/"}"
  log_info "image" "using '$activegate_image' on '$activegate_registry' as activegate image"

  # parse docker config
  entries=$(echo "$pull_secret" | jq -c '.auths | to_entries[]')
  for entry in $entries ; do
    registry=$(echo "$entry" | jq -r '.key')
    username=$(echo "$entry" | jq -r '.value.username')
    password=$(echo "$entry" | jq -r '.value.password')

    check_registry="$container_cli 'curl -u $username:$password --head https://$registry/v2/ -s -o /dev/null'"
    if ! eval "${check_registry}" ; then
      error "registry '$registry' unreachable"
    else
      log_info "image" "registry '$registry' is accessible"
    fi

    log_info "image" "checking images for registry '$registry'"

    # check oneagent image
    check_image="$container_cli 'curl -u $username:$password --head \
      https://$registry/v2/$oneagent_image/manifests/latest -s -o /dev/null'"
    if ! eval "${check_image}" ; then
      error "image '$oneagent_image' on registry '$registry' unreachable"
    else
      log_info "image" "image '$oneagent_image' exists on registry '$registry'"
    fi

    # check activegate image
    check_image="$container_cli 'curl -u $username:$password --head \
      https://$registry/v2/$activegate_image/manifests/latest -s -o /dev/null'"
    if ! eval "${check_image}" ; then
      error "image '$activegate_image' on registry '$registry' unreachable"
    else
      log_info "image" "image '$activegate_image' exists on registry '$registry'"
    fi
  done
}

checkClusterConnection() {
  container_cli="$1"

  log_info "connection" "checking if connection to cluster is valid ..."
  curl_params=(
    -sI
    -o "/dev/null"
    "${api_url}/v1/deployment/installer/agent/unix/default/latest/metainfo"
    "-H" "\"Authorization: Api-Token ${paas_token}\""
  )

  # proxy
  proxy=""
  proxy_secret_name=$("${cli}" get dynakube "${selected_dynakube}" -n "$selected_namespace" --template="{{.spec.proxy.valueFrom}}")
  if [[ "$proxy_secret_name" != "$missing_value" ]]; then
    # get proxy from secret
    encoded_proxy=$("${cli}" get secret "${proxy_secret_name}" -n "$selected_namespace" --template="{{.data.proxy}}")
    proxy=$(echo "$encoded_proxy" | base64 -d)
    log_info "connection" "loading proxy from secret '$proxy_secret_name'"
  else
    # try get proxy from dynakube
    proxyValue=$("${cli}" get dynakube "${selected_dynakube}" -n "$selected_namespace" --template="{{.spec.proxy.value}}")
    if [[ "$proxyValue" != "$missing_value" ]]; then
      proxy=$proxyValue
    fi
  fi

  if [[ "$proxy" != "" ]]; then
    log_info "connection" "using proxy: $proxy"
    curl_params+=("--proxy" "${proxy}")
  else
    log_info "connection" "proxy is not used"
  fi

  # skip cert check
  skip_cert_check=$("${cli}" get dynakube "${selected_dynakube}" -n "$selected_namespace" --template="{{.spec.skipCertCheck}}")
  if [[ "$skip_cert_check" == "true" ]]; then
    log_info "connection" "skipping cert check"
    curl_params+=("--insecure")
  fi

  # trusted ca
  custom_ca_map=$("${cli}" get dynakube "${selected_dynakube}" -n "$selected_namespace" --template="{{.spec.trustedCAs}}")
  if [[ "$custom_ca_map" != "$missing_value" ]]; then
    # get custom certificate from config map and save to file
    certs=$("${cli}" get configmap "${custom_ca_map}" -n "$selected_namespace" --template="{{.data.certs}}")
    cert_path="/tmp/ca.pem"

    log_info "connection" "copying certificate to container ..."
    ca_cmd="$container_cli \"echo '$certs' > $cert_path\""
    if ! eval "$ca_cmd"; then
      error "unable to write custom certificate to container"
    else
      log_info "connection" "custom certificate successfully written to container!"
    fi

    log_info "connection" "using custom certificate in '$cert_path'"
    curl_params+=("--cacert" "$cert_path")
  else
    log_info "connection" "custom certificate is not used"
  fi

  log_info "connection" "trying to access cluster '$api_url' ..."
  connection_cmd="$container_cli \"curl ${curl_params[*]}\""
  if ! eval "${connection_cmd}"; then
    error "unable to connect to cluster"
  else
    log_info "connection" "cluster is accessible"
  fi
}

####### MAIN #######

checkNs
checkDynakube

# choose operator pod to check connection/images
operator_pod=$("${cli}" get pods -n "$selected_namespace" --no-headers -o custom-columns=":metadata.name" | grep dynatrace-operator)
log_info "pod" "using pod '$operator_pod'"
container_cli="${cli} exec ${operator_pod} -- /bin/bash -c"

checkClusterConnection "$container_cli"
checkImagePullable "$container_cli"

# todo: look through support channel for common pitfalls
