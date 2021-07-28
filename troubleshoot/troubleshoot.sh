#!/bin/bash

cli="kubectl" # todo: oc for openshift
missing_value="<no value>"

if ! "${cli}" get ns dynatrace >/dev/null 2>&1; then
  echo "missing namespace 'dynatrace'"
  exit 1
fi

# check dynakube crd exists
echo "checking dynakube ..."

crd="$("${cli}" get dynakube -n dynatrace >/dev/null)"
if [ -z "$crd" ]
then
  echo "dynakube crd exists"
else
  echo "No dynakube crd"
  exit 1
fi

# check dynakube cr exists
names="$("${cli}" get dynakube -n dynatrace -o jsonpath={..metadata.name})"
if [ -z "$names" ]
then
  echo "dynakube cr does not exist"
  exit 1
fi

for name in $names
do
#  echo $name
  dk_name=$name
done
# todo: handle multiple different dynakubes by selecting one

echo "dynakube with name '${dk_name}' selected"

# check api url
echo
echo "checking api url..."

api_url=$("${cli}" get dynakube "${dk_name}" -n dynatrace --template="{{.spec.apiUrl}}")
url_end="${api_url##*/}"
if [ ! "$url_end" = "api" ]
then
  echo "api url has to end on '/api'"
  exit 1
fi
# todo: check for valid url?

# check secret
echo
echo "checking secret for dynakube ..."

# todo: check for different secret name (.spec.tokens)
secret="$("${cli}" get secret "$dk_name" -n dynatrace &> /dev/null)"
if [ -z "$secret" ]
then
  echo "secret exists"
else
  echo "secret with the name '${dk_name}' is missing"
  exit 1
fi

token_names=("apiToken" "paasToken")
paas_token=""
for token_name in "${token_names[@]}"
do
  token=$("${cli}" get secret dynakube -n dynatrace --template="{{.data.${token_name}}}")
  if [ "$token" = "$missing_value" ]
  then
    echo "token '${token_name}' does not exist"
    exit 1
  fi

  if [ "$token_name" = "paasToken" ]
  then
    paas_token=$(echo "$token" | base64 -d)
  fi
done

echo "api token: $paas_token"

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
proxy_map=$("${cli}" get dynakube "${dk_name}" -n dynatrace --template="{{.spec.proxy.valueFrom}}")
if [[ "$proxy_map" != "$missing_value" ]]
then
  # get proxy from secret
  echo "secret key" "$proxy_map"
  encoded_proxy=$("${cli}" get secret "${proxy_map}" -n dynatrace --template="{{.data.proxy}}")
  proxy=$(echo "$encoded_proxy" | base64 -d)
else
  # try get proxy from dynakube
  proxyValue=$("${cli}" get dynakube "${dk_name}" -n dynatrace --template="{{.spec.proxy.value}}")
  if [[ "$proxyValue" != "$missing_value" ]]
  then
    proxy=$proxyValue
  fi
fi

if [[ "$proxy" != "" ]]
then
  echo "using proxy:" "$proxy"
  curl_params+=("--proxy" "${proxy}")
fi

# skip cert check
skip_cert_check=$("${cli}" get dynakube "${dk_name}" -n dynatrace --template="{{.spec.skipCertCheck}}")
if [[ "$skip_cert_check" = "true" ]]
then
  echo "skipping cert check"
  curl_params+=("--insecure")
fi

# trusted ca
custom_ca_map=$("${cli}" get dynakube "${dk_name}" -n dynatrace --template="{{.spec.trustedCAs}}")
if [[ "$custom_ca_map" != "$missing_value" ]]
then
  # get custom certificate from config map and save to file
  certs=$("${cli}" get configmap "${custom_ca_map}" -n dynatrace --template="{{.data.certs}}")
  cert_path="/tmp/ca.pem"
  echo "$certs" > "$cert_path"
  echo "using custom certificate:" "$cert_path"
  curl_params+=("--cacert" "$cert_path")
fi

#curl_command="curl"
#response="$(${curl_command} -sI -o "/dev/null" -X "GET" \
#  "${api_url}/v1/deployment/installer/agent/unix/default/latest/metainfo" \
#  -H "Authorization: Api-Token ${paas_token}")"

if curl "${curl_params[@]}"
then
  echo "no problems found for your setup"
else
  echo "there was a problem with your setup"
fi

# todo: look through support channel for common pitfalls