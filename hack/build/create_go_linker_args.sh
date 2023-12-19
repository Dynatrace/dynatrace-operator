#!/bin/bash

if [ -z "$2" ]
then
  echo "Usage: $0 <version> <commit_hash>"
  exit 1
fi

version=$1
commit=$2

build_date="$(date -u +"%Y-%m-%dT%H:%M:%S+00:00")"
go_linker_args=(
  "-X 'github.com/Dynatrace/dynatrace-operator/pkg/version.Version=${version}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/pkg/version.Commit=${commit}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/pkg/version.BuildDate=${build_date}'"
  "-extldflags=-static -s -w"
)
echo "${go_linker_args[*]}"
