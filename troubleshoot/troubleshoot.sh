#!/bin/bash

set -e

cli="kubectl"
missing_value="<no value>"
selected_dynakube=""
api_url=""
paas_token=""

while [ $# -gt 0 ]; do
  case "$1" in
  --dynakube)
    selected_dynakube="$2"
    shift 2
    ;;
  --oc)
    cli="oc"
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

checkNs() {
  if ! "${cli}" get ns dynatrace >/dev/null 2>&1; then
    log_info "namespace" "missing namespace 'dynatrace'"
    exit 1
  fi
}

checkDynakube() {
  # check dynakube crd exists
  crd="$("${cli}" get dynakube -n dynatrace >/dev/null 2>&1)"
  if [ -z "$crd" ]; then
    log_info "dynakube" "crd exists"
  else
    log_info "dynakube" "crd missing"
    exit 1
  fi

  # check dynakube cr exists
  if [[ -n "$selected_dynakube" ]] ; then
    # dynakube set via parameter
    if ! "${cli}" get dynakube "${selected_dynakube}" -n dynatrace >/dev/null 2>&1 ; then
      log_info "dynakube" "selected dynakube does not exist!"
      exit 1
    fi
  else
    # dynakube not set, check for existing
    names="$("${cli}" get dynakube -n dynatrace -o jsonpath={..metadata.name})"
    if [ -z "$names" ]; then
      log_info "dynakube" "cr does not exist"
      exit 1
    fi

    read -ra names_arr <<<"$names"
    selected_dynakube="${names_arr[0]}"
  fi

  log_info "dynakube" "'${selected_dynakube}' selected"

  log_info "dynakube" "checking api url"
  checkApiUrl

  log_info "dynakube" "checking secret"
  checkSecret

  log_info "dynakube" "'${selected_dynakube}' is valid"
}

checkApiUrl() {
  api_url=$("${cli}" get dynakube "${selected_dynakube}" -n dynatrace --template="{{.spec.apiUrl}}")
  if [ "${api_url##*/}" != "api" ]; then
    log_info "dynakube" "api url has to end on '/api'"
    exit 1
  fi
  # todo: check for valid url?
}

checkSecret() {
  # use dynakube name or tokens value if set
  secret_name="$selected_dynakube"
  tokens=$("${cli}" get dynakube "${selected_dynakube}" -n dynatrace --template="{{.spec.tokens}}")
  if [[ "$tokens" != "$missing_value" ]]; then
    secret_name=$tokens
  fi

  if ! "${cli}" get secret "$secret_name" -n dynatrace &>/dev/null; then
    log_info "dynakube" "secret with the name '${secret_name}' is missing"
    exit 1
  fi

  token_names=("apiToken" "paasToken")
  for token_name in "${token_names[@]}"; do
    token=$("${cli}" get secret "$secret_name" -n dynatrace --template="{{.data.${token_name}}}")
    if [[ "$token" == "$missing_value" ]]; then
      log_info "dynakube" "token '${token_name}' does not exist in secret '${secret_name}'"
      exit 1
    fi

    if [ "$token_name" = "paasToken" ]; then
      paas_token=$(echo "$token" | base64 -d)
    fi
  done
}

checkConnection() {
  operator_pod=$("${cli}" get pods --no-headers -o custom-columns=":metadata.name" | grep dynatrace-operator)
  log_info "connection" "using pod '$operator_pod'"
  container_cli="${cli} exec ${operator_pod} -- /bin/bash -c"

  curl_params=(
    -sI
    -o "/dev/null"
    "${api_url}/v1/deployment/installer/agent/unix/default/latest/metainfo"
    "-H" "\"Authorization: Api-Token ${paas_token}\""
  )

  # proxy
  proxy=""
  proxy_map=$("${cli}" get dynakube "${selected_dynakube}" -n dynatrace --template="{{.spec.proxy.valueFrom}}")
  if [[ "$proxy_map" != "$missing_value" ]]; then
    # get proxy from secret
    encoded_proxy=$("${cli}" get secret "${proxy_map}" -n dynatrace --template="{{.data.proxy}}")
    proxy=$(echo "$encoded_proxy" | base64 -d)
  else
    # try get proxy from dynakube
    proxyValue=$("${cli}" get dynakube "${selected_dynakube}" -n dynatrace --template="{{.spec.proxy.value}}")
    if [[ "$proxyValue" != "$missing_value" ]]; then
      proxy=$proxyValue
    fi
  fi

  if [[ "$proxy" != "" ]]; then
    log_info "connection" "using proxy: $proxy"
    curl_params+=("--proxy" "${proxy}")
  fi

  # skip cert check
  skip_cert_check=$("${cli}" get dynakube "${selected_dynakube}" -n dynatrace --template="{{.spec.skipCertCheck}}")
  if [[ "$skip_cert_check" == "true" ]]; then
    log_info "connection" "skipping cert check"
    curl_params+=("--insecure")
  fi

  # trusted ca
  custom_ca_map=$("${cli}" get dynakube "${selected_dynakube}" -n dynatrace --template="{{.spec.trustedCAs}}")
  if [[ "$custom_ca_map" != "$missing_value" ]]; then
    # get custom certificate from config map and save to file
    certs=$("${cli}" get configmap "${custom_ca_map}" -n dynatrace --template="{{.data.certs}}")
    cert_path="/tmp/ca.pem"

    log_info "connection" "copying certificate to container ..."
    ca_cmd="$container_cli \"echo '$certs' > $cert_path\""
    if ! eval "$ca_cmd" ; then
      log_info "connection" "unable to write custom certificate"
      exit 1
    fi

    log_info "connection" "using custom certificate in '$cert_path'"
    curl_params+=("--cacert" "$cert_path")
  fi

  log_info "connection" "trying to access cluster ..."
  connection_cmd="$container_cli \"curl ${curl_params[*]}\""
  if ! eval "${connection_cmd}"; then
    log_info "connection" "unable to connect to cluster"
    exit 1
  fi
}

####### MAIN #######
log_info "namespace" "checking ..."
checkNs

log_info "dynakube" "checking ..."
checkDynakube

log_info "connection" "checking ..."
checkConnection

# todo: look through support channel for common pitfalls
