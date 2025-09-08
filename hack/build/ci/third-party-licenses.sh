#!/bin/bash
#renovate depName=github.com/google/go-licenses
golang_licenses_version=v2.0.1

# get licenses if no cache exists
if ! [ -d ./third_party_licenses ]; then
  go install github.com/google/go-licenses@${golang_licenses_version} && go-licenses save ./... --save_path third_party_licenses --force
fi
