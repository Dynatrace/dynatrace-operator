#!/bin/bash

set -e

selected_dynakube="dynakube"
selected_namespace="dynatrace"
cli="kubectl"
default_oneagent_image="docker.io/dynatrace/oneagent"

missing_value="<no value>"
api_url=""
paas_token=""
log_section=""

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
    echo "ERROR: unsupported option: '$1'"
    exit 1
    ;;
  esac
done

log_info() {
  printf "[%10s] %s\n" "$log_section" "$1"
}

error() {
  printf "ERROR: %s\n" "$1"
  exit 1
}

checkDependencies() {
  if ! command -v jq &> /dev/null
  then
      error "jq is required to run this script!"
  fi
}

checkNs() {
  log_section="namespace"
  log_info "checking if namespace '${selected_namespace}' exists .."
  if ! "${cli}" get ns "${selected_namespace}" >/dev/null 2>&1; then
    error "missing namespace '${selected_namespace}'"
  else
    log_info "using namespace '${selected_namespace}'"
  fi
}

checkDynakube() {
  log_section="dynakube"
  log_info "checking if Dynakube is configured correctly ..."

  # check dynakube crd exists
  crd="$("${cli}" get dynakube -n "${selected_namespace}" >/dev/null 2>&1)"
  if [[ "$crd" == "" ]]; then
    log_info "CRD for Dynakube exists"
  else
    error "CRD for Dynakube missing"
  fi

  # check selected dynakube exists
  if ! "${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" >/dev/null 2>&1; then
    error "Selected Dynakube '${selected_dynakube}' does not exist"
  fi

  log_info "using '${selected_dynakube}'"

  checkApiUrl
  checkSecret
  checkCustomPullSecret

  log_info "'${selected_dynakube}' is valid"
}

checkApiUrl() {
  log_info "checking if api url is valid..."

  api_url=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" --template="{{.spec.apiUrl}}")
  if [[ "${api_url##*/}" != "api" ]]; then
    error "api url has to end on '/api'"
  else
    log_info "api url correctly ends on '/api'"
  fi

  log_info "api url is valid"
}

checkSecret() {
  log_info "checking if secret is valid ..."

  # use dynakube name or tokens value if set
  secret_name="$selected_dynakube"
  tokens=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" --template="{{.spec.tokens}}")
  if [[ "$tokens" != "$missing_value" ]]; then
    secret_name=$tokens
  fi

  if ! "${cli}" get secret "$secret_name" -n "${selected_namespace}" &>/dev/null; then
    error "secret with the name '${secret_name}' is missing"
  else
    log_info "secret '${secret_name}' exists"
  fi

  token_names=("apiToken" "paasToken")
  for token_name in "${token_names[@]}"; do
    token=$("${cli}" get secret "$secret_name" -n "${selected_namespace}" --template="{{.data.${token_name}}}")
    if [[ "$token" == "$missing_value" ]]; then
      error "token '${token_name}' does not exist in secret '${secret_name}'"
    else
      log_info "secret token '${token_name}' exists"
    fi

    if [[ "$token_name" == "paasToken" ]]; then
      paas_token=$(echo "$token" | base64 -d)
    fi
  done
}

checkImmutableImage() {
  type="$1"

  use_immutable_image=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" \
    --template="{{.spec.${type}.useImmutableImage}}")
  if [[ "$use_immutable_image" == "true" ]] ; then
    error "unable to use immutable image on ${type} without a private registry (custom pull secret)"
  fi
}

checkCustomPullSecret() {
  pull_secret_name=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" \
    --template="{{.spec.customPullSecret}}")
  if [[ "${pull_secret_name}" == "" ]] || [[ "${pull_secret_name}" == "${missing_value}" ]]; then
    log_info "custom pull secret not used"

    # private registry required for immutable image
    checkImmutableImage "classicFullStack"
    checkImmutableImage "infraMonitoring"
  else
    # custom pull secret is set, check secret exists
    if ! "${cli}" get secret "${pull_secret_name}" -n "${selected_namespace}" >/dev/null 2>&1; then
      error "secret '${pull_secret_name}' used for pull secret is missing"
    else
      log_info "pull secret '${pull_secret_name}' exists"
    fi
  fi
}

getImage() {
  type="$1"

  if [[ "${type}" == "oneAgent" ]] ; then
    # oneagent uses docker.io by default
    image="${default_oneagent_image}"
  else
    # activegate is not published and uses the cluster registry by default
    api_url=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" --template="{{.spec.apiUrl}}")
    image="${api_url#*//}"
    image="${image%/*}/linux/activegate"
  fi

  dynakube_image=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" --template="{{.spec.${type}.image}}")
  if [[ "${dynakube_image}" != "" && "$dynakube_image" != "$missing_value" ]]; then
    image="${dynakube_image}"
  else
    # use version if image not set
    dynakube_version=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" --template="{{.spec.${type}.version}}")
    if [[ "${dynakube_version}" != "" && "$dynakube_version" != "$missing_value" ]]; then
      image+=":${dynakube_version}"
    fi
  fi

  echo "${image}"
}

checkImagePullable() {
  container_cli="$1"

  log_section="image"
  log_info "checking if images are pullable ..."

  # load pull secret
  custom_pull_secret_name=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" --template="{{.spec.customPullSecret}}")
  if [[ "${custom_pull_secret_name}" != "" && "${custom_pull_secret_name}" != "${missing_value}" ]] ; then
    pull_secret_name="$custom_pull_secret_name"
  else
    pull_secret_name="$selected_dynakube-pull-secret"
  fi
  log_info "using pull secret '$pull_secret_name'"

  pull_secret_encoded=$("${cli}" get secret "${pull_secret_name}" -n "${selected_namespace}" -o "jsonpath={.data['\.dockerconfigjson']}")
  pull_secret="$(echo "${pull_secret_encoded}" | base64 -d)"

  # load used images (default or custom)
  dynakube_oneagent_image=$(getImage "oneAgent")
  dynakube_activegate_image=$(getImage "activeGate")

  # split images into registry and image name
  oneagent_registry="${dynakube_oneagent_image%%/*}"
  oneagent_image="${dynakube_oneagent_image##"$oneagent_registry/"}"
  IFS=':'
  read -ra split_image <<< "$oneagent_image"
  if [[ ${#split_image[@]} == 1 ]] ; then
    # no version set, default to latest
    oneagent_version="latest"

    log_info "using latest image version"
  else
    oneagent_image=${split_image[0]}
    oneagent_version=${split_image[1]}

    log_info "using custom image version"
  fi
  IFS=' '
  log_info "using '$oneagent_image' on '$oneagent_registry' with version '$oneagent_version' as oneagent image"

  activegate_registry="${dynakube_activegate_image%%/*}"
  activegate_image="${dynakube_activegate_image##"$activegate_registry/"}"
  log_info "using '$activegate_image' on '$activegate_registry' as activegate image"

  # parse docker config
  oneagent_image_works=false
  activegate_image_works=false
  entries=$(echo "$pull_secret" | jq -c '.auths | to_entries[]')
  for entry in $entries ; do
    registry=$(echo "$entry" | jq -r '.key')
    username=$(echo "$entry" | jq -r '.value.username')
    password=$(echo "$entry" | jq -r '.value.password')

    check_registry="$container_cli 'curl -u $username:$password --head https://$registry/v2/ -s -o /dev/null'"
    if ! eval "${check_registry}" ; then
      error "registry '$registry' unreachable"
    else
      log_info "registry '$registry' is accessible"
    fi

    log_info "checking images for registry '$registry'"

    # check oneagent image
    check_image="$container_cli 'curl -u $username:$password --head \
      https://$registry/v2/$oneagent_image/manifests/latest -s -o /dev/null -w %{http_code}'"
    image_response_code=$(eval "${check_image}")
    if [[ "$image_response_code" != "200" ]] ; then
      log_info "image '$oneagent_image' with version '$oneagent_version' not found on registry '$registry'"
    else
      log_info "image '$oneagent_image' with version '$oneagent_version' exists on registry '$registry'"
      if [[ "$registry" == "$oneagent_registry" ]] ; then
        oneagent_image_works=true
      fi
    fi

    # check activegate image
    check_image="$container_cli 'curl -u $username:$password --head \
      https://$registry/v2/$activegate_image/manifests/latest -s -o /dev/null -w %{http_code}'"
    image_response_code=$(eval "${check_image}")
    if [[ "$image_response_code" != "200" ]] ; then
      log_info "image '$activegate_image' not found on registry '$registry'"
    else
      log_info "image '$activegate_image' exists on registry '$registry'"
      if [[ "$registry" == "$activegate_registry" ]] ; then
        activegate_image_works=true
      fi
    fi
  done

  if [[ "$oneagent_image_works" == "true" ]] ; then
    log_info "oneagent image '$dynakube_oneagent_image' found"
  else
    if [[ "$oneagent_registry" == "docker.io" ]] ; then
      # get auth token with pull access for docker hub registry
      token=$(
        curl --silent \
        "https://auth.docker.io/token?service=registry.docker.io&scope=repository:$oneagent_image:pull" \
        | jq -r '.token'
      )

      # check selected image exists on docker hub
      dockerio_image_request="$container_cli 'curl --head --header \"Authorization: Bearer ${token}\" \
        https://registry-1.docker.io/v2/$oneagent_image/manifests/$oneagent_version -s -o /dev/null -w %{http_code}'"

      if [[ "$(eval "$dockerio_image_request")" == "200" ]] ; then
        log_info "'$oneagent_image' with version '$oneagent_version' exists on docker.io registry"
      else
        error "oneagent image '$oneagent_image' with version '$oneagent_version' not found on docker.io registry"
      fi
    else
      error "oneagent image '$dynakube_oneagent_image' with version '$oneagent_version' missing."
    fi
  fi

  if [[ "$activegate_image_works" == "true" ]] ; then
    log_info "activegate image '$dynakube_activegate_image' found"
  else
    error "activegate image '$dynakube_activegate_image' missing"
  fi
}

checkConnection() {
  container_cli="$1"

  log_section="connection"
  log_info "checking if connection to cluster is valid ..."
  curl_params=(
    -sI
    -o "/dev/null"
    "${api_url}/v1/deployment/installer/agent/unix/default/latest/metainfo"
    "-H" "\"Authorization: Api-Token ${paas_token}\""
  )

  # proxy
  proxy=""
  proxy_secret_name=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" --template="{{.spec.proxy.valueFrom}}")
  if [[ "$proxy_secret_name" != "" && "$proxy_secret_name" != "$missing_value" ]]; then
    # get proxy from secret
    encoded_proxy=$("${cli}" get secret "${proxy_secret_name}" -n "${selected_namespace}" --template="{{.data.proxy}}")
    proxy=$(echo "$encoded_proxy" | base64 -d)
    log_info "loading proxy from secret '$proxy_secret_name'"
  else
    # try get proxy from dynakube
    proxyValue=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" --template="{{.spec.proxy.value}}")
    if [[ "$proxyValue" != "$missing_value" ]]; then
      proxy=$proxyValue
    fi
  fi

  if [[ "$proxy" != "" ]]; then
    log_info "using proxy: $proxy"
    curl_params+=("--proxy" "${proxy}")
  else
    log_info "proxy is not used"
  fi

  # skip cert check
  skip_cert_check=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" --template="{{.spec.skipCertCheck}}")
  if [[ "$skip_cert_check" == "true" ]]; then
    log_info "skipping cert check"
    curl_params+=("--insecure")
  fi

  # trusted ca
  custom_ca_map=$("${cli}" get dynakube "${selected_dynakube}" -n "${selected_namespace}" --template="{{.spec.trustedCAs}}")
  if [[ "$custom_ca_map" != "" && "$custom_ca_map" != "$missing_value" ]]; then
    # get custom certificate from config map and save to file
    certs=$("${cli}" get configmap "${custom_ca_map}" -n "${selected_namespace}" --template="{{.data.certs}}")
    cert_path="/tmp/ca.pem"

    log_info "copying certificate to container ..."
    ca_cmd="$container_cli \"echo '$certs' > $cert_path\""
    if ! eval "$ca_cmd"; then
      error "unable to write custom certificate to container"
    else
      log_info "custom certificate successfully written to container!"
    fi

    log_info "using custom certificate in '$cert_path'"
    curl_params+=("--cacert" "$cert_path")
  else
    log_info "custom certificate is not used"
  fi

  log_info "trying to access tenant '$api_url' ..."
  connection_cmd="$container_cli \"curl ${curl_params[*]}\""
  if ! eval "${connection_cmd}"; then
    error "unable to connect to tenant"
  else
    log_info "tenant is accessible"
  fi
}

####### MAIN #######

checkDependencies

checkNs
checkDynakube

# choose operator pod to check connection/images
operator_pod=$("${cli}" get pods -n "${selected_namespace}" --no-headers -o custom-columns=":metadata.name" | grep dynatrace-operator)
log_info "using pod '$operator_pod'"
container_cli="${cli} exec ${operator_pod} -n ${selected_namespace} -- /bin/bash -c"

checkConnection "$container_cli"
checkImagePullable "$container_cli"

echo
echo "No known issues found with the dynatrace-operator installation!"
