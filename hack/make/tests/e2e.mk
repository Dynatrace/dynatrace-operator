## Runs e2e tests
test/e2e: manifests/branch test/e2e/cloudnative test/e2e/applicationmonitoring test/e2e/activegate test/e2e/supportarchive

## Runs ActiveGate e2e test only
test/e2e/activegate: manifests
	go test -v -tags e2e -timeout 20m -count=1 -failfast ./test/activegate

## Runs CloudNative e2e test only
test/e2e/cloudnative: manifests
	go test -v -tags e2e -timeout 10m -count=1 -failfast ./test/cloudnative

## Runs CloudNative e2e test only
test/e2e/classic: manifests
	go test -v -tags e2e -timeout 10m -count=1 -failfast ./test/classic

## Runs CloudNative istio e2e test only
test/e2e/cloudnative/istio: manifests
	go test -v -tags e2e -count=1 -failfast ./test/cloudnativeistio

## Runs CloudNative proxy e2e test only
test/e2e/cloudnative/proxy: manifests
	go test -v -tags e2e -count=1 -failfast ./test/cloudnative/proxy

## Runs Application Monitoring e2e test only
test/e2e/applicationmonitoring: manifests
	go test -v -tags e2e -count=1 -failfast ./test/appmon

## Runs Application Monitoring e2e test only
test/e2e/supportarchive:
	go test -v -tags e2e -count=1 -failfast ./test/support_archive
