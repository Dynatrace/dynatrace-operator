## Runs e2e tests
test/e2e:
	go test -v -tags e2e -timeout 30m ./test/...

## Runs ActiveGate e2e test only
test/e2e/activegate:
	go test -v -tags e2e -count=1 ./test/activegate

## Runs CloudNative e2e test only
test/e2e/cloudnative:
	go test -v -tags e2e -count=1 ./test/cloudnative
