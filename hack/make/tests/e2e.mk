## Start a test and skip TEARDOWN steps if it fails
test/e2e/%/debug:
	@make SKIPCLEANUP="--fail-fast" $(@D)

## Run standard, istio and release e2e tests
test/e2e:
	RC=0; \
	make test/e2e/standard  || RC=1; \
	make test/e2e/istio  || RC=1; \
	make test/e2e/release || RC=1; \
	exit $$RC

## Run standard e2e test only
test/e2e/standard: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 200m -count=1 ./test/scenarios/standard -args --skip-labels "name=cloudnative-network-zone" $(SKIPCLEANUP)

## Run istio e2e test only
test/e2e/istio: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 200m -count=1 ./test/scenarios/istio -args $(SKIPCLEANUP)

## Run release e2e test only
test/e2e/release: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/release -args $(SKIPCLEANUP)

## Runs ActiveGate e2e test only
test/e2e/activegate: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=activegate-default" $(SKIPCLEANUP)

## Runs ActiveGate proxy e2e test only
test/e2e/activegate/proxy: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --labels "name=activegate-proxy" $(SKIPCLEANUP)

## Runs ClassicFullStack e2e test only
test/e2e/classic: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=classic" $(SKIPCLEANUP)

## Runs ClassicFullStack switch mode e2e test only
test/e2e/classic/switchmodes: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=classic-to-cloudnative" $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e test only
test/e2e/cloudnative/codemodules: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=cloudnative-codemodules-image" $(SKIPCLEANUP)

## Runs CloudNative codemodules-with-proxy e2e test only
test/e2e/cloudnative/codemodules-with-proxy: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --labels "name=codemodules-with-proxy" $(SKIPCLEANUP)

## Runs CloudNative codemodules-with-proxy-custom-ca e2e test only
test/e2e/cloudnative/codemodules-with-custom-ca: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --labels "name=codemodules-with-proxy-custom-ca" $(SKIPCLEANUP)

## Runs CloudNative automatic injection disabled e2e test only
test/e2e/cloudnative/disabledautoinjection: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=cloudnative-disabled-auto-inject" $(SKIPCLEANUP)

## Runs CloudNative default e2e test only
test/e2e/cloudnative/default: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=cloudnative-default" $(SKIPCLEANUP)

## Runs CloudNative istio e2e test only
test/e2e/cloudnative/istio: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --labels "name=cloudnative-istio" $(SKIPCLEANUP)

## Runs CloudNative network problem e2e test only
test/e2e/cloudnative/resilience: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --labels "name=cloudnative-csi-resilience" $(SKIPCLEANUP)

test/e2e/cloudnative/network-zone: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=cloudnative-network-zone" $(SKIPCLEANUP)

## Runs CloudNative proxy e2e test only
test/e2e/cloudnative/proxy: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --labels "name=cloudnative-proxy" $(SKIPCLEANUP)

## Runs Classic/CloudNative mode switching tests
test/e2e/cloudnative/switchmodes: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=cloudnative-to-classic" $(SKIPCLEANUP)

## Runs CloudNative upgrade e2e test only
test/e2e/cloudnative/upgrade: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/release -args --labels "name=cloudnative-upgrade" $(SKIPCLEANUP)

## Runs Application Monitoring data-ingest e2e test only
test/e2e/applicationmonitoring/dataingest: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=app-data-ingest"  $(SKIPCLEANUP)

## Runs Application Monitoring label versio detection e2e test only
test/e2e/applicationmonitoring/labelversion: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=app-label-version"  $(SKIPCLEANUP)

## Runs Application Monitoring readonly csi-volume e2e test only
test/e2e/applicationmonitoring/readonlycsivolume: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=app-read-only-csi-volume"  $(SKIPCLEANUP)

## Runs Application Monitoring without CSI e2e test only
test/e2e/applicationmonitoring/withoutcsi: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=app-without-csi" $(SKIPCLEANUP)

## Runs public registry images e2e test only
test/e2e/publicregistry: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=public-registry-images" $(SKIPCLEANUP)

## Runs SupportArchive e2e test only
test/e2e/supportarchive: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=support-archive" $(SKIPCLEANUP)

## Runs Edgeconnect e2e test only
test/e2e/edgeconnect: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=edgeconnect-install" $(SKIPCLEANUP)

## Runs e2e tests on gke-autopilot
test/e2e/gke-autopilot: manifests/kubernetes/gke-autopilot
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=app-data-ingest,name=app-read-only-csi-volume,name=app-read-only-csi-volume,name=app-without-csi,name=activegate-default" $(SKIPCLEANUP)

## Runs Application Monitoring data-ingest e2e test only on gke-autopilot
test/e2e/gke-autopilot/applicationmonitoring/dataingest: manifests/kubernetes/gke-autopilot
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=app-data-ingest"  $(SKIPCLEANUP)

## Runs Application Monitoring label versio detection e2e test only on gke-autopilot
test/e2e/gke-autopilot/applicationmonitoring/labelversion: manifests/kubernetes/gke-autopilot
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=app-label-version"  $(SKIPCLEANUP)

## Runs Application Monitoring readonly csi-volume e2e test only on gke-autopilot
test/e2e/gke-autopilot/applicationmonitoring/readonlycsivolume: manifests/kubernetes/gke-autopilot
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=app-read-only-csi-volume" $(SKIPCLEANUP)

## Runs Application Monitoring without CSI e2e test only on gke-autopilot
test/e2e/gke-autopilot/applicationmonitoring/withoutcsi: manifests/kubernetes/gke-autopilot
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --labels "name=app-without-csi" $(SKIPCLEANUP)
