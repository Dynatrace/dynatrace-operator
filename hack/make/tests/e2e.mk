GOTESTCMD:=go test

## Start a test and save the result to an xml file
test/e2e/%/publish:
	@make GOTESTCMD='gotestsum --format standard-verbose --junitfile results/$(notdir $(@D)).xml --' $(@D)

## Start a test and skip TEARDOWN steps if it fails
test/e2e/%/debug:
	@make SKIPCLEANUP="--fail-fast" $(@D)

## Run standard, no-csi, istio and release e2e tests
test/e2e:
	RC=0; \
	make test/e2e/standard  || RC=1; \
	make test/e2e/no-csi || RC=1; \
	make test/e2e/istio  || RC=1; \
	make test/e2e/release || RC=1; \
	exit $$RC

## Run standard, no-csi, istio and release e2e tests with /publish
test/e2e-publish:
	RC=0; \
	make test/e2e/standard/publish || RC=1; \
	make test/e2e/no-csi/publish || RC=1; \
	make test/e2e/istio/publish || RC=1; \
	make test/e2e/release/publish || RC=1; \
	exit $$RC

## Run standard e2e test only
test/e2e/standard: manifests/crd/helm
	$(GOTESTCMD) -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 200m -count=1 ./test/scenarios/standard -args $(SKIPCLEANUP)

## Run istio e2e test only
test/e2e/istio: manifests/crd/helm
	$(GOTESTCMD) -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 200m -count=1 ./test/scenarios/istio -args $(SKIPCLEANUP)

## Run no-csi e2e test only
test/e2e/no-csi: manifests/crd/helm
	$(GOTESTCMD) -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 200m -count=1 ./test/scenarios/no_csi -args $(SKIPCLEANUP)

## Run release e2e test only
test/e2e/release: manifests/crd/helm
	$(GOTESTCMD) -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1 ./test/scenarios/release -args $(SKIPCLEANUP)

## Runs ActiveGate e2e test only
test/e2e/activegate: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "activegate" $(SKIPCLEANUP)

## Runs ActiveGate proxy e2e test only
test/e2e/activegate/proxy: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --feature "activegate" $(SKIPCLEANUP)

## Runs ClassicFullStack e2e test only
test/e2e/classic: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "classic" $(SKIPCLEANUP)

## Runs ClassicFullStack switch mode e2e test only
test/e2e/classic/switchmodes: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --feature "classic-to-cloudnative" $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e test only
test/e2e/cloudnative/codemodules: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --feature "cloudnative-codemodules-image" $(SKIPCLEANUP)

## Runs CloudNative codemodules-with-proxy e2e test only
test/e2e/cloudnative/codemodules-with-proxy: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --feature "codemodules-with-proxy-no-certs" $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e test with proxy and AG custom certificate
test/e2e/cloudnative/codemodules-with-proxy-and-ag-cert: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --feature "codemodules-with-proxy-and-ag-cert" $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e test with proxy and automatically created AG certificate
test/e2e/cloudnative/codemodules-with-proxy-and-auto-ag-cert: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --feature "codemodules-with-proxy-and-auto-ag-cert" $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e test with proxy and AG custom certificates
test/e2e/cloudnative/codemodules-with-proxy-custom-ca-ag-cert: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --feature "codemodules-with-proxy-custom-ca-ag-cert" $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e test with proxy and automatically created AG certificates
test/e2e/cloudnative/codemodules-with-proxy-custom-ca-auto-ag-cert: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --feature "codemodules-with-proxy-custom-ca-auto-ag-cert" $(SKIPCLEANUP)

## Runs CloudNative automatic injection disabled e2e test only
test/e2e/cloudnative/disabledautoinjection: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --feature "cloudnative-disabled-auto-inject" $(SKIPCLEANUP)

## Runs CloudNative default e2e test only
test/e2e/cloudnative/default: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --feature "cloudnative" $(SKIPCLEANUP)

## Runs CloudNative istio e2e test only
test/e2e/cloudnative/istio: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --feature "cloudnative" $(SKIPCLEANUP)

## Runs CloudNative network problem e2e test only
test/e2e/cloudnative/resilience: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/istio -args --feature "cloudnative-csi-resilience" $(SKIPCLEANUP)

## Runs Classic/CloudNative mode switching tests
test/e2e/cloudnative/switchmodes: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --feature "cloudnative-to-classic" $(SKIPCLEANUP)

## Runs CloudNative upgrade e2e test only
test/e2e/cloudnative/upgrade: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/release -args --feature "cloudnative-upgrade" $(SKIPCLEANUP)

## Runs Application Monitoring metadata-enrichment e2e test only
test/e2e/applicationmonitoring/metadataenrichment: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "metadata-enrichment"  $(SKIPCLEANUP)

## Runs Application Monitoring label versio detection e2e test only
test/e2e/applicationmonitoring/labelversion: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "label-version"  $(SKIPCLEANUP)

## Runs Application Monitoring readonly csi-volume e2e test only
test/e2e/applicationmonitoring/readonlycsivolume: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --feature "app-read-only-csi-volume"  $(SKIPCLEANUP)

## Runs Application Monitoring without CSI e2e test only
test/e2e/applicationmonitoring/withoutcsi: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "app-monitoring-without-csi" $(SKIPCLEANUP)

## Runs Application Monitoring bootstrapper with CSI e2e test only
test/e2e/applicationmonitoring/bootstrapper-csi: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --feature "node-image-pull-with-csi" $(SKIPCLEANUP)

## Runs Application Monitoring bootstrapper with no CSI e2e test only
test/e2e/applicationmonitoring/bootstrapper-no-csi: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "node-image-pull-with-no-csi" $(SKIPCLEANUP)

## Runs public registry images e2e test only
test/e2e/publicregistry: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --feature "public-registry-images" $(SKIPCLEANUP)

## Runs SupportArchive e2e test only
test/e2e/supportarchive: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --feature "support-archive" $(SKIPCLEANUP)

## Runs Edgeconnect e2e test only
test/e2e/edgeconnect: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "edgeconnect-.*" $(SKIPCLEANUP)

## Runs e2e tests on gke-autopilot
test/e2e/gke-autopilot: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/standard -args --feature "app-metadata-enrichment|app-read-only-csi-volume|app-read-only-csi-volume|app-without-csi|activegate" $(SKIPCLEANUP)

## Runs extensions related e2e tests
test/e2e/extensions: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "extensions-components-rollout" $(SKIPCLEANUP)

## Runs LogMonitoring related e2e tests
test/e2e/logmonitoring: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "logmonitoring-components-rollout" $(SKIPCLEANUP)

## Runs Host Monitoring without CSI e2e test only
test/e2e/hostmonitoring/withoutcsi: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "host-monitoring-without-csi" $(SKIPCLEANUP)

## Runs CloudNative default e2e test only
test/e2e/cloudnative/withoutcsi: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "cloudnative" $(SKIPCLEANUP)

## Runs TelemetryIngest related e2e tests
test/e2e/telemetryingest: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "telemetryingest-.*" $(SKIPCLEANUP)

test/e2e/telemetryingest/public-active-gate: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "telemetryingest-with-public-ag-components-rollout" $(SKIPCLEANUP)

test/e2e/telemetryingest/local-active-gate-and-cleanup: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "telemetryingest-with-local-active-gate-component-rollout-and-cleanup-after-disable" $(SKIPCLEANUP)

test/e2e/telemetryingest/otel-collector-endpoint-tls: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "telemetryingest-with-otel-collector-endpoint-tls" $(SKIPCLEANUP)

test/e2e/telemetryingest/otel-collector-config-udpate: manifests/crd/helm
	go test -v -tags "$(shell ./hack/build/create_go_build_tags.sh true)" -timeout 20m -count=1  ./test/scenarios/no_csi -args --feature "telemetryingest-configuration-update" $(SKIPCLEANUP)
