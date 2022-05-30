## Install kuttl if it is missing
test/kuttl/install:
	hack/e2e/install-kuttl.sh

## Checks if the environment variables APIURL, APITOKEN and PAASTOKEN are set for kuttl tests
test/kuttl/check-env-vars:
	hack/do_env_variables_exist.sh "APIURL APITOKEN PAASTOKEN"

## Runs Activegate specific kuttl tests
test/kuttl/activegate: test/kuttl/check-env-vars
	kubectl kuttl test --config kuttl/activegate/testsuite.yaml

## Runs OneAgent specific kuttl tests
test/kuttl/oneagent: test/kuttl/check-env-vars
	kubectl kuttl test --config kuttl/oneagent/oneagent-test.yaml

## Runs all available kuttl tests
test/kuttl: test/kuttl/activegate test/kuttl/oneagent