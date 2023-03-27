## Runs e2e tests
test/e2e:
	./hack/e2e/run_all.sh

## Runs ActiveGate e2e test only
test/e2e/activegate: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/activegate/basic

## Runs ActiveGate proxy e2e test only
test/e2e/activegate/proxy: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/activegate/proxy

## Runs CloudNative e2e test only
test/e2e/cloudnative: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 30m -count=1 ./test/scenarios/cloudnative/basic

## Runs ClassicFullStack e2e test only
## TODO: rename after proper implementation of cleanup step
test/e2e/zz_classic: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/classic

## Runs CloudNative istio e2e test only
test/e2e/cloudnative/istio: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/cloudnative/istio

## Runs CloudNative proxy e2e test only
test/e2e/cloudnative/proxy: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -count=1 ./test/scenarios/cloudnative/proxy

## Runs CloudNative network problem e2e test only
test/e2e/cloudnative/network: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/cloudnative/network

## Runs Application Monitoring e2e test only
test/e2e/applicationmonitoring: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -count=1 ./test/scenarios/applicationmonitoring

## Runs SupportArchive e2e test only
test/e2e/supportarchive: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -count=1 ./test/scenarios/support_archive

## synthetic monitoring
test/e2e/synthetic: manifests/crd/helm
	go test -v -tags e2e -timeout 25m -count=1 ./test/scenarios/synthetic/monolocation
