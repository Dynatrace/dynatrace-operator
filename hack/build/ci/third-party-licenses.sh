#!/bin/bash
#renovate depName=github.com/google/go-licenses
GOLANG_LICENSES_VERSION ?= v2.0.1

# get licenses if no cache exists
if ! [ -d ./third_party_licenses ]; then
  go install github.com/google/go-licenses/v2@${GOLANG_LICENSES_VERSION} && go-licenses save ./... --save_path third_party_licenses --force
fi
