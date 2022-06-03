#!/bin/bash

# get licenses if no cache exists
if ! [ -d ./third_party_licenses ]; then
  go get github.com/google/go-licenses && go-licenses save ./... --save_path third_party_licenses --force
fi

# fetch dependencies
go get -d ./...
ls -la $HOME/go/pkg/mod
cp -r $HOME/go/pkg/mod .
