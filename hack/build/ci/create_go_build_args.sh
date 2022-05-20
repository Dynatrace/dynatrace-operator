#!/bin/bash

version=$1
commit=$2

build_date="$(date -u +"%Y-%m-%dT%H:%M:%S+00:00")"
go_build_args=(
  "-ldflags=-X 'github.com/Dynatrace/dynatrace-operator/src/version.Version=${version}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/src/version.Commit=${commit}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/src/version.BuildDate=${build_date}'"
  "-s -w"
)
echo "${go_build_args[*]}"
