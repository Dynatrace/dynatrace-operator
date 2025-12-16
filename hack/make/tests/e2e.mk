GOTESTFLAGS := -v -count 1 -tags "$(shell ./hack/build/create_go_build_tags.sh true)"
GOTESTCMD := go test $(GOTESTFLAGS)

## Start a test and save the result to an xml file
test/e2e/%/publish:
	@make GOTESTCMD='gotestsum --format standard-verbose --junitfile results/$(notdir $(@D)).xml -- $(GOTESTFLAGS)' $(@D)

## Start a test and skip TEARDOWN steps if it fails
test/e2e/%/debug:
	@make SKIPCLEANUP="-args --fail-fast" $(@D)

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
	$(GOTESTCMD) -timeout 200m ./test/scenarios/standard $(SKIPCLEANUP)

## Run istio e2e test only
test/e2e/istio: manifests/crd/helm
	$(GOTESTCMD) -timeout 200m ./test/scenarios/istio $(SKIPCLEANUP)

## Run no-csi e2e test only
test/e2e/no-csi: manifests/crd/helm
	$(GOTESTCMD) -timeout 200m ./test/scenarios/nocsi $(SKIPCLEANUP)

## Run release e2e test only
test/e2e/release: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/release $(SKIPCLEANUP)

## Runs ActiveGate e2e test only
test/e2e/activegate: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "activegate" $(SKIPCLEANUP)

## Runs ActiveGate proxy e2e test only
test/e2e/activegate/proxy: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/istio -run "activegate" $(SKIPCLEANUP)

## Runs ClassicFullStack e2e test only
test/e2e/classic: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "classic"  $(SKIPCLEANUP)

## Runs ClassicFullStack switch mode e2e test only
test/e2e/classic/switchmodes: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "classic_to_cloudnative"  $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e test only
test/e2e/cloudnative/codemodules: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "cloudnative_codemodules_image" $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e migrate to image only
test/e2e/cloudnative/codemodules-migrate-to-image: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "codemodules_migrate_to_image" $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e migrate to node-image-pull only
test/e2e/cloudnative/codemodules-migrate-to-node-image-pull: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "codemodules_migrate_to_node_image_pull" $(SKIPCLEANUP)

## Runs CloudNative codemodules-with-proxy e2e test only
test/e2e/cloudnative/codemodules-with-proxy: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/istio -run "codemodules_with_proxy_no_certs" $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e test with proxy and AG custom certificate
test/e2e/cloudnative/codemodules-with-proxy-and-ag-cert: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/istio -run "codemodules_with_proxy_and_ag_cert" $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e test with proxy and automatically created AG certificate
test/e2e/cloudnative/codemodules-with-proxy-and-auto-ag-cert: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/istio -run "codemodules_with_proxy_and_auto_ag_cert" $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e test with proxy and AG custom certificates
test/e2e/cloudnative/codemodules-with-proxy-custom-ca-ag-cert: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/istio -run "codemodules_with_proxy_custom_ca_ag_cert"  $(SKIPCLEANUP)

## Runs CloudNative codemodules e2e test with proxy and automatically created AG certificates
test/e2e/cloudnative/codemodules-with-proxy-custom-ca-auto-ag-cert: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/istio -run "codemodules_with_proxy_custom_ca_auto_ag_cert" $(SKIPCLEANUP)

## Runs CloudNative automatic injection disabled e2e test only
test/e2e/cloudnative/disabledautoinjection: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "cloudnative_disabled_auto_inject" $(SKIPCLEANUP)

## Runs CloudNative default e2e test only
test/e2e/cloudnative/default: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "cloudnative" $(SKIPCLEANUP)

## Runs CloudNative istio e2e test only
test/e2e/cloudnative/istio: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/istio -run "cloudnative" $(SKIPCLEANUP)

## Runs CloudNative network problem e2e test only
test/e2e/cloudnative/resilience: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/istio -run "cloudnative_csi_resilience" $(SKIPCLEANUP)

## Runs Classic/CloudNative mode switching tests
test/e2e/cloudnative/switchmodes: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "cloudnative_to_classic" $(SKIPCLEANUP)

## Runs CloudNative upgrade e2e test only
test/e2e/cloudnative/upgrade: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/release -run "cloudnative_upgrade" $(SKIPCLEANUP)

## Runs extensions upgrade e2e test only
test/e2e/extensions/upgrade: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/release -run "extensions_upgrade" $(SKIPCLEANUP)

## Runs DatabaseExecutor related e2e tests
test/e2e/extensions/dbexecutor: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "extensions_db_executor" $(SKIPCLEANUP)

## Runs Application Monitoring metadata-enrichment e2e test only
test/e2e/applicationmonitoring/metadataenrichment: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "metadata_enrichment" $(SKIPCLEANUP)

## Runs Application Monitoring otlp-exporter-configuration e2e test only
test/e2e/applicationmonitoring/otlpexporterconfiguration: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "otlp_exporter_configuration" $(SKIPCLEANUP)

## Runs Application Monitoring label version detection e2e test only
test/e2e/applicationmonitoring/labelversion: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "label_version" $(SKIPCLEANUP)

## Runs Application Monitoring readonly csi-volume e2e test only
test/e2e/applicationmonitoring/readonlycsivolume: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "app_read_only_csi_volume" $(SKIPCLEANUP)

## Runs Application Monitoring without CSI e2e test only
test/e2e/applicationmonitoring/withoutcsi: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "app_monitoring_without_csi" $(SKIPCLEANUP)

## Runs Application Monitoring bootstrapper with CSI e2e test only
test/e2e/applicationmonitoring/bootstrapper-csi: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "node_image_pull_with_csi" $(SKIPCLEANUP)

## Runs Application Monitoring bootstrapper with no CSI e2e test only
test/e2e/applicationmonitoring/bootstrapper-no-csi: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "node_image_pull_with_no_csi" $(SKIPCLEANUP)

## Runs public registry images e2e test only
test/e2e/publicregistry: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "public_registry_images" $(SKIPCLEANUP)

## Runs SupportArchive e2e test only
test/e2e/supportarchive: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "support_archive" $(SKIPCLEANUP)

## Runs Edgeconnect e2e tests
test/e2e/edgeconnect: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "edgeconnect" $(SKIPCLEANUP)

## Runs Edgeconnect e2e base test cases
test/e2e/edgeconnect/normal: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "TestNoCSI_edgeconnect_install" $(SKIPCLEANUP)

## Runs Edgeconnect e2e proxy test cases
test/e2e/edgeconnect/proxy: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m  ./test/scenarios/nocsi -run "TestNoCSI_edgeconnect_install_proxy" $(SKIPCLEANUP)

## Runs e2e tests on gke-autopilot
test/e2e/gke-autopilot: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/standard -run "app_metadata_enrichment|app_read_only_csi_volume|app_read_only_csi_volume|app_without_csi|activegate" $(SKIPCLEANUP)

## Runs extensions related e2e tests
test/e2e/extensions: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "extensions" $(SKIPCLEANUP)

## Runs LogMonitoring related e2e tests
test/e2e/logmonitoring: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "logmonitoring.*" $(SKIPCLEANUP)

test/e2e/logmonitoring/optionalscopes: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "logmonitoring_with_optional_scopes.*" $(SKIPCLEANUP)

## Runs Host Monitoring without CSI e2e test only
test/e2e/hostmonitoring/withoutcsi: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "host_monitoring_without_csi" $(SKIPCLEANUP)

## Runs CloudNative default e2e test only
test/e2e/cloudnative/withoutcsi: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "cloudnative" $(SKIPCLEANUP)

## Runs TelemetryIngest related e2e tests
test/e2e/telemetryingest: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "telemetryingest_.*" $(SKIPCLEANUP)

test/e2e/telemetryingest/public-active-gate: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "telemetryingest_w_public_ag" $(SKIPCLEANUP)

test/e2e/telemetryingest/local-active-gate-and-cleanup: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "telemetryingest_w_local_ag_and_cleanup_after" $(SKIPCLEANUP)

test/e2e/telemetryingest/otel-collector-endpoint-tls: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "telemetryingest_w_otel_collector_endpoint_tls" $(SKIPCLEANUP)

test/e2e/telemetryingest/otel-collector-config-udpate: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "telemetryingest_configuration_update" $(SKIPCLEANUP)

test/e2e/kspm: manifests/crd/helm
	$(GOTESTCMD) -timeout 20m ./test/scenarios/nocsi -run "kspm" $(SKIPCLEANUP)
