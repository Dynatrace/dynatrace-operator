#!/usr/bin/env bash

tests=$(make help | grep -oE "prerequisites\/\S*" | grep -v "%\/debug")
result=0
for test in $tests
do
    make "$test"
    result=$((result+$?))
done
exit $result
