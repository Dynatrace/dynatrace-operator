## Runs e2e tests
test/e2e: manifests/branch test/e2e/cloudnative test/e2e/applicationmonitoring test/e2e/activegate

## Runs ActiveGate e2e test only
test/e2e/activegate:
	go test -v -tags e2e -timeout 20m -count=1 -failfast ./test/activegate

## Runs CloudNative e2e test only
test/e2e/cloudnative:
	go test -v -tags e2e -count=1 -failfast ./test/cloudnative

## Runs CloudNative istio e2e test only
test/e2e/cloudnative/istio:
	go test -v -tags e2e -count=1 -failfast ./test/cloudnativeistio

## Runs CloudNative proxy e2e test only
test/e2e/cloudnative/proxy:
	go test -v -tags e2e -count=1 -failfast ./test/cloudnative/proxy

## Runs Application Monitoring e2e test only
test/e2e/applicationmonitoring:
	go test -v -tags e2e -count=1 -failfast ./test/appmon
