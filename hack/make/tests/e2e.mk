## Runs e2e tests
test/e2e: test/e2e/activegate test/e2e/cloudnative test/e2e/appmon

## Runs ActiveGate e2e test only
test/e2e/activegate:
	go test -v -tags e2e -timeout 20m -count=1 -failfast ./test/activegate

## Runs CloudNative e2e test only
test/e2e/cloudnative:
	go test -v -tags e2e -count=1 -failfast ./test/cloudnative

## Runs Application Monitoring e2e test only
test/e2e/appmon:
	go test -v -tags e2e -count=1 -failfast ./test/appmon