#!/bin/bash

cli="kubectl" # todo: oc for openshift
missing_value="<no value>"
selected_dynakube=""
api_url=""
paas_token=""

log() {
  printf "[%s] %s\n" "$1" "$2"
}

checkNs() {
  if ! "${cli}" get ns dynatrace >/dev/null 2>&1; then
    log "namespace" "missing namespace 'dynatrace'"
    exit 1
  fi
}

checkDynakube() {
  # check dynakube crd exists
  crd="$("${cli}" get dynakube -n dynatrace >/dev/null)"
  if [ -z "$crd" ]; then
    log "dynakube" "crd exists"
  else
    log "dynakube" "crd missing"
    exit 1
  fi

  # check dynakube cr exists
  names="$("${cli}" get dynakube -n dynatrace -o jsonpath={..metadata.name})"
  if [ -z "$names" ]; then
    log "dynakube" "cr does not exist"
    exit 1
  fi

  for name in $names; do
    #  echo $name
    selected_dynakube=$name
  done
  # todo: handle multiple different dynakubes by selecting one

  log "dynakube" "'${selected_dynakube}' selected"

  log "dynakube" "checking api url"
  checkApiUrl
  log "dynakube" "api url valid"

  log "dynakube" "checking secret"
  checkSecret
  log "dynakube" "secret valid"
}

checkApiUrl() {
  api_url=$("${cli}" get dynakube "${selected_dynakube}" -n dynatrace --template="{{.spec.apiUrl}}")
  url_end="${api_url##*/}"
  if [ ! "$url_end" = "api" ]; then
    log "dynakube" "api url has to end on '/api'"
    exit 1
  fi
  # todo: check for valid url?
}

checkSecret() {
  # todo: check for different secret name (.spec.tokens)
  secret="$("${cli}" get secret "$selected_dynakube" -n dynatrace &>/dev/null)"
  if [ -n "$secret" ]; then
    log "dynakube" "secret with the name '${selected_dynakube}' is missing"
    exit 1
  fi

  token_names=("apiToken" "paasToken")
  for token_name in "${token_names[@]}"; do
    token=$("${cli}" get secret dynakube -n dynatrace --template="{{.data.${token_name}}}")
    if [ "$token" = "$missing_value" ]; then
      log "dynakube" "token '${token_name}' does not exist in secret '$selected_dynakube'"
      exit 1
    fi

    if [ "$token_name" = "paasToken" ]; then
      paas_token=$(echo "$token" | base64 -d)
    fi
  done
}

checkConnection() {
  # todo: get operator pod
  # todo: connect to container and use api url to make connection to cluster: with proxy, trustedCA, ...

  curl_params=(
    -sI
    -o "/dev/null"
    "${api_url}/v1/deployment/installer/agent/unix/default/latest/metainfo"
    "-H" "Authorization: Api-Token ${paas_token}"
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
    log "connection" "using proxy: $proxy"
    curl_params+=("--proxy" "${proxy}")
  fi

  # skip cert check
  skip_cert_check=$("${cli}" get dynakube "${selected_dynakube}" -n dynatrace --template="{{.spec.skipCertCheck}}")
  if [[ "$skip_cert_check" == "true" ]]; then
    log "connection" "skipping cert check"
    curl_params+=("--insecure")
  fi

  # trusted ca
  custom_ca_map=$("${cli}" get dynakube "${selected_dynakube}" -n dynatrace --template="{{.spec.trustedCAs}}")
  if [[ "$custom_ca_map" != "$missing_value" ]]; then
    # get custom certificate from config map and save to file
    certs=$("${cli}" get configmap "${custom_ca_map}" -n dynatrace --template="{{.data.certs}}")
    cert_path="/tmp/ca.pem"
    echo "$certs" >"$cert_path"

    log "connection" "using custom certificate: $cert_path"
    curl_params+=("--cacert" "$cert_path")
  fi

  log "connection" "trying to access cluster ..."
  if ! curl "${curl_params[@]}"; then
    log "connection" "unable to connect to cluster"
    exit 1
  fi
}

####### MAIN #######
log "namespace" "checking ..."
checkNs
log "namespace" "valid"

log "dynakube" "checking ..."
checkDynakube
log "dynakube" "valid"

log "connection" "checking ..."
checkConnection
log "connection" "valid"

# todo: look through support channel for common pitfalls
