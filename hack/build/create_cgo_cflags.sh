#!/bin/bash

cgo_cflags=(
  "-O2"
  "-Wno-return-local-addr"
)
echo "${cgo_cflags[*]}"
