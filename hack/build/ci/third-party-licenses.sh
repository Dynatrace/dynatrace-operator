#!/bin/bash

# get licenses if no cache exists
if ! [ -d ./third_party_licenses ]; then
  go install github.com/google/go-licenses@v1.6.0 && go-licenses save ./... --save_path third_party_licenses --force
fi

