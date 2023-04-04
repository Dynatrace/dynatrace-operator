## Start a test and skip TEARDOWN steps if it fails
test/e2e/%/debug:
	@make SKIPCLEANUP="-args --fail-fast" $(@D)

## Runs e2e tests
test/e2e:
	./hack/e2e/run_all.sh

## Runs ActiveGate e2e test only
test/e2e/activegate: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/activegate/basic $(SKIPCLEANUP)

## Runs ActiveGate proxy e2e test only
test/e2e/activegate/proxy: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/activegate/proxy $(SKIPCLEANUP)

## Runs CloudNative e2e test only
test/e2e/cloudnative: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 30m -count=1 ./test/scenarios/cloudnative/basic $(SKIPCLEANUP)

## Runs ClassicFullStack e2e test only
## TODO: rename after proper implementation of cleanup step
test/e2e/zz_classic: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/classic $(SKIPCLEANUP)

## Runs CloudNative istio e2e test only
test/e2e/cloudnative/istio: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/cloudnative/istio $(SKIPCLEANUP)

## Runs CloudNative proxy e2e test only
test/e2e/cloudnative/proxy: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/cloudnative/proxy $(SKIPCLEANUP)

## Runs CloudNative network problem e2e test only
test/e2e/cloudnative/network: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/cloudnative/network $(SKIPCLEANUP)

## Runs Application Monitoring e2e test only
test/e2e/applicationmonitoring: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -count=1 ./test/scenarios/applicationmonitoring $(SKIPCLEANUP)

## Runs SupportArchive e2e test only
test/e2e/supportarchive: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -count=1 ./test/scenarios/support_archive $(SKIPCLEANUP)
