#!/bin/bash

cgo_flags=(
  "-O2"
  "-Wno-return-local-addr"
)
echo "${cgo_cflags[*]}"
