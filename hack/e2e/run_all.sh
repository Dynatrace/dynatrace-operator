#!/usr/bin/env bash

tests=$(make help | grep -oE "test\/e2e\/\S*")
for test in $tests
do
    make "$test"
done

