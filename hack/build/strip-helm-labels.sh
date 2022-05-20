#!/usr/bin/env bash

tmpfile=$(mktemp)

for file in $*; do
  if [ -f "${file}" ]; then
    grep -v 'app.kubernetes.io/managed-by' "$file" > "$tmpfile"
    grep -v 'helm.sh' "$tmpfile" > "$file"
  fi
done

rm $tmpfile
