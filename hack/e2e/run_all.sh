#!/usr/bin/env bash

tests=$(make help | grep -oE "test\/e2e\/\S*")
result=0
for test in $tests
do
    make "$test"
    result=$((result+$?))
done
exit $result
