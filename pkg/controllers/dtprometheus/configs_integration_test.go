// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"encoding/json"
	"maps"
	"net"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// The rendered configs are asserted semantically here, not as strings: the exact YAML is already
// pinned by the golden files in each component package (gateway/scraper/targetallocator testdata).
// What those golden files cannot show is whether the names and ports inside a config actually
// match the Service and container objects that end up in the cluster, which is what these tests
// check.

// collectorConfig is the shape shared by the gateway and scraper OTel Collector configs.
type collectorConfig struct {
	Receivers  map[string]json.RawMessage `json:"receivers"`
	Processors map[string]json.RawMessage `json:"processors"`
	Exporters  map[string]json.RawMessage `json:"exporters"`
	Extensions map[string]json.RawMessage `json:"extensions"`
	Connectors map[string]json.RawMessage `json:"connectors"`
	Service    struct {
		Extensions []string `json:"extensions"`
		Pipelines  map[string]struct {
			Receivers  []string `json:"receivers"`
			Processors []string `json:"processors"`
			Exporters  []string `json:"exporters"`
		} `json:"pipelines"`
	} `json:"service"`
}

// targetAllocatorConfig mirrors the fields of targetallocator.Config that the wiring depends on.
type targetAllocatorConfig struct {
	ListenAddr         string                `json:"listen_addr"`
	AllocationStrategy string                `json:"allocation_strategy"`
	FilterStrategy     string                `json:"filter_strategy"`
	CollectorNamespace string                `json:"collector_namespace"`
	CollectorSelector  *metav1.LabelSelector `json:"collector_selector"`
	PrometheusCR       struct {
		Enabled                         bool                  `json:"enabled"`
		ScrapeInterval                  string                `json:"scrape_interval"`
		PodMonitorSelector              *metav1.LabelSelector `json:"pod_monitor_selector"`
		ServiceMonitorSelector          *metav1.LabelSelector `json:"service_monitor_selector"`
		ScrapeConfigSelector            *metav1.LabelSelector `json:"scrape_config_selector"`
		ProbeSelector                   *metav1.LabelSelector `json:"probe_selector"`
		PodMonitorNamespaceSelector     *metav1.LabelSelector `json:"pod_monitor_namespace_selector"`
		ServiceMonitorNamespaceSelector *metav1.LabelSelector `json:"service_monitor_namespace_selector"`
		ScrapeConfigNamespaceSelector   *metav1.LabelSelector `json:"scrape_config_namespace_selector"`
		ProbeNamespaceSelector          *metav1.LabelSelector `json:"probe_namespace_selector"`
	} `json:"prometheus_cr"`
}

// scraperReceiver is the prometheus receiver in target-allocator mode.
type scraperReceiver struct {
	TargetAllocator struct {
		Endpoint    string `json:"endpoint"`
		Interval    string `json:"interval"`
		CollectorID string `json:"collector_id"`
	} `json:"target_allocator"`
}

// loadBalancingExporter is the scraper's exporter towards the gateway pool.
type loadBalancingExporter struct {
	Resolver struct {
		K8s struct {
			Service string `json:"service"`
			Ports   []int  `json:"ports"`
		} `json:"k8s"`
	} `json:"resolver"`
}

// otlpHTTPExporter is the gateway's exporter towards the Dynatrace tenant.
type otlpHTTPExporter struct {
	Endpoint string `json:"endpoint"`
	Auth     struct {
		Authenticator string `json:"authenticator"`
	} `json:"auth"`
	TLS struct {
		CAFile string `json:"ca_file"`
	} `json:"tls"`
}

// bearerTokenAuth is the gateway extension that reads the tenant token from disk.
type bearerTokenAuth struct {
	Scheme   string `json:"scheme"`
	Filename string `json:"filename"`
}

func (f *fixture) gatewayConfig(t *testing.T) collectorConfig {
	t.Helper()

	return unmarshalConfig[collectorConfig](t, f.configMap(t, f.dtp.Gateway().GetStatefulSetName()), "relay")
}

func (f *fixture) scraperConfig(t *testing.T) collectorConfig {
	t.Helper()

	return unmarshalConfig[collectorConfig](t, f.configMap(t, f.dtp.Scraper().GetDeploymentName()), "scraper")
}

func (f *fixture) targetAllocatorConfig(t *testing.T) targetAllocatorConfig {
	t.Helper()

	return unmarshalConfig[targetAllocatorConfig](t,
		f.configMap(t, f.dtp.TargetAllocator().GetDeploymentName()), "targetallocator.yaml")
}

func unmarshalConfig[T any](t *testing.T, cm *corev1.ConfigMap, key string) T {
	t.Helper()

	raw, ok := cm.Data[key]
	require.Truef(t, ok, "ConfigMap %s has no key %s (keys: %v)", cm.Name, key, slices.Sorted(maps.Keys(cm.Data)))

	var parsed T

	require.NoErrorf(t, yaml.Unmarshal([]byte(raw), &parsed), "ConfigMap %s key %s is not valid YAML", cm.Name, key)

	return parsed
}

func unmarshalInto[T any](t *testing.T, raw json.RawMessage, what string) T {
	t.Helper()

	var parsed T

	require.NoErrorf(t, json.Unmarshal(raw, &parsed), "cannot parse %s", what)

	return parsed
}

// testConfigMaps checks each rendered config on its own: valid, internally consistent, and
// carrying the values the spec asked for.
func testConfigMaps(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "configmaps")
	f.reconcileSuccessfully(t)

	t.Run("gateway collector config", func(t *testing.T) {
		cfg := f.gatewayConfig(t)
		assertCollectorConfigConsistent(t, cfg)

		require.Contains(t, cfg.Service.Pipelines, "metrics")
		assert.Equal(t, []string{"otlp"}, cfg.Service.Pipelines["metrics"].Receivers)
		assert.Equal(t, []string{"otlp_http"}, cfg.Service.Pipelines["metrics"].Exporters)
		assert.Equal(t,
			[]string{"memory_limiter", "metric_start_time", "cumulative_to_delta", "k8s_attributes", "transform"},
			cfg.Service.Pipelines["metrics"].Processors)

		exporter := unmarshalInto[otlpHTTPExporter](t, cfg.Exporters["otlp_http"], "gateway otlp_http exporter")
		assert.Equal(t, testAPIURL+"/v2/otlp", exporter.Endpoint, "the tenant endpoint comes from the DynaKube apiUrl")
		assert.Equal(t, "bearertokenauth", exporter.Auth.Authenticator)
		assert.Empty(t, exporter.TLS.CAFile, "no custom CA is configured on the DynaKube")
	})

	t.Run("scraper collector config", func(t *testing.T) {
		cfg := f.scraperConfig(t)
		assertCollectorConfigConsistent(t, cfg)

		require.Contains(t, cfg.Service.Pipelines, "metrics")
		assert.Equal(t, []string{"prometheus"}, cfg.Service.Pipelines["metrics"].Receivers)
		assert.Equal(t, []string{"load_balancing"}, cfg.Service.Pipelines["metrics"].Exporters)

		receiver := unmarshalInto[scraperReceiver](t, cfg.Receivers["prometheus"], "scraper prometheus receiver")
		assert.Equal(t, "1m0s", receiver.TargetAllocator.Interval, "the CRD default for targetsPollInterval is 60s")
		// Each scraper pod must identify itself to the target allocator with a unique, stable id,
		// and the only thing that qualifies is its own pod name.
		assert.Equal(t, "${env:MY_POD_NAME}", receiver.TargetAllocator.CollectorID)
		f.assertEnvIsDefined(t, f.dtp.Scraper().GetDeploymentName(), "scraper", "MY_POD_NAME")
	})

	t.Run("target allocator config", func(t *testing.T) {
		cfg := f.targetAllocatorConfig(t)

		assert.Equal(t, "consistent-hashing", cfg.AllocationStrategy,
			"consistent hashing keeps targets on the same scraper when the pool scales")
		assert.Equal(t, "relabel-config", cfg.FilterStrategy)
		assert.True(t, cfg.PrometheusCR.Enabled)
		assert.Equal(t, "1m0s", cfg.PrometheusCR.ScrapeInterval, "the CRD default for scrapeInterval is 60s")

		// All four Prometheus Operator CRD kinds must be selected the same way, otherwise a
		// scrapeCRSelector would silently apply to some kinds only.
		defaultSelector := &metav1.LabelSelector{MatchLabels: map[string]string{"prometheus.dynatrace.com": "true"}}
		assert.Equal(t, defaultSelector, cfg.PrometheusCR.PodMonitorSelector)
		assert.Equal(t, defaultSelector, cfg.PrometheusCR.ServiceMonitorSelector)
		assert.Equal(t, defaultSelector, cfg.PrometheusCR.ScrapeConfigSelector)
		assert.Equal(t, defaultSelector, cfg.PrometheusCR.ProbeSelector)

		assert.Nil(t, cfg.PrometheusCR.PodMonitorNamespaceSelector, "all namespaces by default")
		assert.Nil(t, cfg.PrometheusCR.ServiceMonitorNamespaceSelector)
		assert.Nil(t, cfg.PrometheusCR.ScrapeConfigNamespaceSelector)
		assert.Nil(t, cfg.PrometheusCR.ProbeNamespaceSelector)
	})

	t.Run("a custom scrapeCRSelector reaches every CRD kind", func(t *testing.T) {
		selector := &metav1.LabelSelector{MatchLabels: map[string]string{"team": "edp"}}
		nsSelector := &metav1.LabelSelector{MatchLabels: map[string]string{"monitored": "true"}}
		f.dtp.Spec.TargetAllocator.ScrapeCRSelector = selector
		f.dtp.Spec.TargetAllocator.ScrapeCRNamespaceSelector = nsSelector
		f.updateDTPrometheus(t)
		f.reconcileSuccessfully(t)

		cfg := f.targetAllocatorConfig(t)
		for name, got := range map[string]*metav1.LabelSelector{
			"pod_monitor":     cfg.PrometheusCR.PodMonitorSelector,
			"service_monitor": cfg.PrometheusCR.ServiceMonitorSelector,
			"scrape_config":   cfg.PrometheusCR.ScrapeConfigSelector,
			"probe":           cfg.PrometheusCR.ProbeSelector,
		} {
			assert.Equal(t, selector, got, "%s_selector", name)
		}

		assert.Equal(t, nsSelector, cfg.PrometheusCR.PodMonitorNamespaceSelector)
		assert.Equal(t, nsSelector, cfg.PrometheusCR.ServiceMonitorNamespaceSelector)
		assert.Equal(t, nsSelector, cfg.PrometheusCR.ScrapeConfigNamespaceSelector)
		assert.Equal(t, nsSelector, cfg.PrometheusCR.ProbeNamespaceSelector)
	})

	t.Run("a custom CA on the DynaKube reaches the gateway exporter and the pod", func(t *testing.T) {
		integrationtests.CreateKubernetesObject(t, clt, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "custom-cas", Namespace: f.ns},
			Data:       map[string]string{"certs": "-----BEGIN CERTIFICATE-----"},
		})
		f.dk.Spec.TrustedCAs = "custom-cas"
		require.NoError(t, clt.Update(t.Context(), f.dk))
		f.reconcileSuccessfully(t)

		exporter := unmarshalInto[otlpHTTPExporter](t, f.gatewayConfig(t).Exporters["otlp_http"], "gateway exporter")
		require.NotEmpty(t, exporter.TLS.CAFile)

		// The path in the config must be a path the pod actually has mounted.
		sts := f.gatewayStatefulSet(t)
		container := containerByName(t, &sts.Spec.Template, "gateway")
		assertPathIsMounted(t, container, exporter.TLS.CAFile)

		for _, volume := range sts.Spec.Template.Spec.Volumes {
			f.assertVolumeSourceResolves(t, volume)
		}
	})
}

// assertCollectorConfigConsistent checks the OTel Collector invariants that make a config
// loadable: every component a pipeline references is defined, and every defined component is
// actually used somewhere. A stale definition is dead config; a missing one crashes the collector
// on startup, which envtest would never notice because it never runs the pod.
func assertCollectorConfigConsistent(t *testing.T, cfg collectorConfig) {
	t.Helper()

	usedReceivers := map[string]bool{}
	usedProcessors := map[string]bool{}
	usedExporters := map[string]bool{}

	require.NotEmpty(t, cfg.Service.Pipelines)

	for name, pipeline := range cfg.Service.Pipelines {
		assertDefined(t, cfg.Receivers, pipeline.Receivers, "receiver of pipeline "+name, usedReceivers)
		assertDefined(t, cfg.Processors, pipeline.Processors, "processor of pipeline "+name, usedProcessors)
		assertDefined(t, cfg.Exporters, pipeline.Exporters, "exporter of pipeline "+name, usedExporters)
	}

	assertAllUsed(t, cfg.Receivers, usedReceivers, "receiver")
	assertAllUsed(t, cfg.Processors, usedProcessors, "processor")
	assertAllUsed(t, cfg.Exporters, usedExporters, "exporter")

	for _, extension := range cfg.Service.Extensions {
		assert.Containsf(t, cfg.Extensions, extension, "service.extensions references undefined extension %s", extension)
	}

	// The gateway's bearertokenauth is referenced by the exporter's authenticator rather than by a
	// pipeline, so extensions are only checked in the "defined" direction.
	assert.Empty(t, cfg.Connectors, "no component uses connectors today")
}

func assertDefined(t *testing.T, defined map[string]json.RawMessage, used []string, what string, seen map[string]bool) {
	t.Helper()

	for _, name := range used {
		assert.Containsf(t, defined, name, "undefined %s: %s", what, name)
		seen[name] = true
	}
}

func assertAllUsed(t *testing.T, defined map[string]json.RawMessage, used map[string]bool, kind string) {
	t.Helper()

	for name := range defined {
		assert.Truef(t, used[name], "%s %s is defined but no pipeline uses it", kind, name)
	}
}

func assertPathIsMounted(t *testing.T, container corev1.Container, path string) {
	t.Helper()

	for _, mount := range container.VolumeMounts {
		if strings.HasPrefix(path, strings.TrimSuffix(mount.MountPath, "/")+"/") {
			return
		}
	}

	assert.Failf(t, "path is not mounted", "%s is referenced in the config but no volumeMount covers it", path)
}

func (f *fixture) assertEnvIsDefined(t *testing.T, workload, containerName, envName string) {
	t.Helper()

	deploy := f.deployment(t, workload)
	container := containerByName(t, &deploy.Spec.Template, containerName)

	for _, env := range container.Env {
		if env.Name == envName {
			return
		}
	}

	assert.Failf(t, "env not defined", "the config expands ${env:%s} but the container does not define it", envName)
}

// testWiring is the cross-component test: it verifies that the names and ports the configs point
// at match the Service and container objects that actually exist, so the whole path
// scraper -> target allocator and scraper -> gateway -> Dynatrace is consistent.
func testWiring(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "wiring")
	f.reconcileSuccessfully(t)

	t.Run("scraper reaches the target allocator", func(t *testing.T) {
		receiver := unmarshalInto[scraperReceiver](t, f.scraperConfig(t).Receivers["prometheus"], "prometheus receiver")

		host, port := splitEndpoint(t, receiver.TargetAllocator.Endpoint)
		svc := f.service(t, f.dtp.TargetAllocator().GetDeploymentName())

		assert.Equal(t, svc.Name+"."+svc.Namespace, host,
			"the configured target allocator host must be the target allocator Service")

		svcPort := servicePortByNumber(t, svc, port)
		assert.Equal(t, "http-port", svcPort.Name, "the scraper talks plain HTTP to the target allocator")

		// ... and that Service port must land on a port the allocator container actually opens,
		// on the address the allocator is configured to listen on.
		taTemplate, _ := targetAllocatorPodSpec(t, f)
		container := containerByName(t, taTemplate, "targetallocator")
		containerPort := containerPortByName(t, container, svcPort.TargetPort.StrVal)

		listenPort := listenAddrPort(t, f.targetAllocatorConfig(t).ListenAddr)
		assert.Equal(t, listenPort, containerPort.ContainerPort,
			"the allocator listens on a different port than the one the Service targets")
	})

	t.Run("target allocator finds the scraper pods", func(t *testing.T) {
		cfg := f.targetAllocatorConfig(t)
		scraperDeploy := f.deployment(t, f.dtp.Scraper().GetDeploymentName())

		assert.Equal(t, scraperDeploy.Namespace, cfg.CollectorNamespace)
		require.NotNil(t, cfg.CollectorSelector)
		assert.Equal(t, scraperDeploy.Spec.Selector.MatchLabels, cfg.CollectorSelector.MatchLabels,
			"the allocator's collector selector must match the scraper Deployment's pods")

		for key, value := range cfg.CollectorSelector.MatchLabels {
			assert.Equal(t, value, scraperDeploy.Spec.Template.Labels[key])
		}
	})

	t.Run("scraper reaches the gateway", func(t *testing.T) {
		exporter := unmarshalInto[loadBalancingExporter](t,
			f.scraperConfig(t).Exporters["load_balancing"], "load_balancing exporter")

		svc := f.service(t, f.dtp.Gateway().GetStatefulSetName())
		assert.Equal(t, svc.Name+"."+svc.Namespace, exporter.Resolver.K8s.Service,
			"the load balancing resolver must point at the gateway Service")
		require.Len(t, exporter.Resolver.K8s.Ports, 1)

		svcPort := servicePortByNumber(t, svc, exporter.Resolver.K8s.Ports[0])
		assert.Equal(t, "otlp", svcPort.Name)

		gwTemplate, _ := gatewayPodSpec(t, f)
		container := containerByName(t, gwTemplate, "gateway")
		containerPort := containerPortByName(t, container, svcPort.TargetPort.StrVal)

		// The gateway's OTLP receiver has to be listening on exactly that container port.
		receiverPort := grpcReceiverPort(t, f.gatewayConfig(t))
		assert.Equal(t, receiverPort, containerPort.ContainerPort,
			"the gateway's OTLP receiver listens on a different port than the one the Service targets")
	})

	t.Run("gateway reaches the tenant", func(t *testing.T) {
		cfg := f.gatewayConfig(t)

		exporter := unmarshalInto[otlpHTTPExporter](t, cfg.Exporters["otlp_http"], "otlp_http exporter")
		assert.True(t, strings.HasPrefix(exporter.Endpoint, testAPIURL), "endpoint %s", exporter.Endpoint)

		auth := unmarshalInto[bearerTokenAuth](t, cfg.Extensions[exporter.Auth.Authenticator], "bearertokenauth extension")
		assert.Equal(t, "Api-Token", auth.Scheme)

		// The token must be read from a file the pod really mounts, from the DynaKube token
		// Secret, and never be inlined into the config.
		gwTemplate, _ := gatewayPodSpec(t, f)
		container := containerByName(t, gwTemplate, "gateway")
		assertPathIsMounted(t, container, auth.Filename)
		f.assertTokenFileComesFromSecret(t, gwTemplate, auth.Filename)
	})
}

// assertTokenFileComesFromSecret resolves the token file path back to the projected Secret volume
// it is served from, and checks it is the DynaKube token Secret's data-ingest key.
func (f *fixture) assertTokenFileComesFromSecret(t *testing.T, tmpl *corev1.PodTemplateSpec, path string) {
	t.Helper()

	container := containerByName(t, tmpl, "gateway")

	for _, mount := range container.VolumeMounts {
		relative, ok := strings.CutPrefix(path, strings.TrimSuffix(mount.MountPath, "/")+"/")
		if !ok {
			continue
		}

		for _, volume := range tmpl.Spec.Volumes {
			if volume.Name != mount.Name || volume.Projected == nil {
				continue
			}

			assertProjectedSecretServesPath(t, f.dk.Tokens(), volume.Projected, relative)

			return
		}
	}

	assert.Failf(t, "token file has no source", "no projected volume serves %s", path)
}

func assertProjectedSecretServesPath(t *testing.T, secretName string, projected *corev1.ProjectedVolumeSource, relative string) {
	t.Helper()

	for _, source := range projected.Sources {
		if source.Secret == nil {
			continue
		}

		for _, item := range source.Secret.Items {
			if item.Path != relative {
				continue
			}

			assert.Equal(t, secretName, source.Secret.Name, "the token must come from the DynaKube token Secret")
			assert.Equal(t, token.DataIngestKey, item.Key)

			return
		}
	}

	assert.Failf(t, "token file has no source", "no projected Secret item produces %s", relative)
}

func splitEndpoint(t *testing.T, endpoint string) (string, int) {
	t.Helper()

	trimmed := strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

	host, portString, err := net.SplitHostPort(trimmed)
	require.NoErrorf(t, err, "cannot split endpoint %s", endpoint)

	port, err := strconv.Atoi(portString)
	require.NoError(t, err)

	return host, port
}

func listenAddrPort(t *testing.T, listenAddr string) int32 {
	t.Helper()

	_, portString, err := net.SplitHostPort(listenAddr)
	require.NoErrorf(t, err, "cannot split listen address %s", listenAddr)

	port, err := strconv.ParseInt(portString, 10, 32)
	require.NoError(t, err)

	return int32(port)
}

// grpcReceiverPort pulls the port out of the gateway's otlp/grpc receiver endpoint, which is
// written as "${env:MY_POD_IP}:<port>".
func grpcReceiverPort(t *testing.T, cfg collectorConfig) int32 {
	t.Helper()

	receiver := unmarshalInto[struct {
		Protocols struct {
			GRPC struct {
				Endpoint string `json:"endpoint"`
			} `json:"grpc"`
		} `json:"protocols"`
	}](t, cfg.Receivers["otlp"], "gateway otlp receiver")

	endpoint := receiver.Protocols.GRPC.Endpoint

	// The host part is an unexpanded "${env:MY_POD_IP}" placeholder, which itself contains a
	// colon, so the port is whatever follows the last one.
	separator := strings.LastIndex(endpoint, ":")
	require.Positivef(t, separator, "endpoint %s has no port", endpoint)

	port, err := strconv.ParseInt(endpoint[separator+1:], 10, 32)
	require.NoError(t, err)

	return int32(port)
}

func servicePortByNumber(t *testing.T, svc *corev1.Service, port int) corev1.ServicePort {
	t.Helper()

	for _, svcPort := range svc.Spec.Ports {
		if int(svcPort.Port) == port {
			return svcPort
		}
	}

	require.Failf(t, "no such service port", "Service %s does not expose port %d", svc.Name, port)

	return corev1.ServicePort{}
}

func containerPortByName(t *testing.T, container corev1.Container, name string) corev1.ContainerPort {
	t.Helper()

	for _, port := range container.Ports {
		if port.Name == name {
			return port
		}
	}

	require.Failf(t, "no such container port", "container %s does not expose a port named %s", container.Name, name)

	return corev1.ContainerPort{}
}

// testSecrets covers the only Secret in play: the DynaKube token Secret, which the controller
// reads but never copies.
// testSecrets covers the only Secret in play. Each subtest gets its own fixture: they mutate the
// DynaKube and create Secrets whose t.Cleanup would otherwise pull the ground out from under the
// next subtest.
func testSecrets(t *testing.T, clt client.Client) {
	t.Run("no token value is ever written in plain text", func(t *testing.T) {
		f := newFixture(t, clt, "secrets-leak")
		f.reconcileSuccessfully(t)

		f.assertNoTokenLeak(t)
	})

	t.Run("rotating the token does not roll any pod", func(t *testing.T) {
		// The token is mounted as a projected directory (not a subPath) precisely so the kubelet
		// refreshes it in place and the bearertokenauth extension picks the new value up. Nothing
		// derives from its content, so no hash may change and no pod may restart.
		f := newFixture(t, clt, "secrets-rotation")
		f.reconcileSuccessfully(t)

		before := f.resourceVersions(t)
		beforeHashes := f.configHashes(t)

		secret := f.tokenSecret(t)
		secret.Data[token.DataIngestKey] = []byte("dt0c01.ROTATEDDATAINGESTTOKENVALUE")
		require.NoError(t, clt.Update(t.Context(), secret))

		f.reconcileSuccessfully(t)

		assert.Equal(t, before, f.resourceVersions(t))
		assert.Equal(t, beforeHashes, f.configHashes(t))
		f.assertNoTokenLeak(t)
	})

	t.Run("a renamed token secret is followed", func(t *testing.T) {
		f := newFixture(t, clt, "secrets-rename")
		f.reconcileSuccessfully(t)

		integrationtests.CreateKubernetesObject(t, clt, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "other-tokens", Namespace: f.ns},
			Data: map[string][]byte{
				token.APIKey:        []byte(testAPIToken),
				token.PaaSKey:       []byte(testPaaSToken),
				token.DataIngestKey: []byte(testDataIngestToken),
			},
		})
		f.dk.Spec.Tokens = "other-tokens"
		require.NoError(t, clt.Update(t.Context(), f.dk))

		f.reconcileSuccessfully(t)

		sts := f.gatewayStatefulSet(t)
		for _, volume := range sts.Spec.Template.Spec.Volumes {
			f.assertVolumeSourceResolves(t, volume)
		}

		assert.Equal(t, "other-tokens", projectedSecretName(t, sts.Spec.Template.Spec.Volumes, "dt-token"))
	})

	t.Run("proxy credentials are referenced, never inlined", func(t *testing.T) {
		f := newFixture(t, clt, "secrets-proxy")
		f.reconcileSuccessfully(t)

		f.assertProxySecretIsReferenced(t, clt)
	})
}

func projectedSecretName(t *testing.T, volumes []corev1.Volume, volumeName string) string {
	t.Helper()

	for _, volume := range volumes {
		if volume.Name != volumeName || volume.Projected == nil {
			continue
		}

		for _, source := range volume.Projected.Sources {
			if source.Secret != nil {
				return source.Secret.Name
			}
		}
	}

	require.Failf(t, "no projected secret", "volume %s has no projected Secret source", volumeName)

	return ""
}

// assertNoTokenLeak walks everything the operator wrote into the namespace, plus the DTPrometheus
// status, and fails if a token value shows up in plain text anywhere. Secret values may only be
// consumed through a secretKeyRef or a Secret volume.
func (f *fixture) assertNoTokenLeak(t *testing.T) {
	t.Helper()

	secrets := []string{testAPIToken, testPaaSToken, testDataIngestToken}

	for _, want := range expectedManagedObjects(f.dtp) {
		obj := f.getManagedObject(t, want)
		haystack := renderForLeakCheck(t, obj)

		for _, secret := range secrets {
			assert.NotContainsf(t, haystack, secret, "%s leaks a token value in plain text", want)
		}
	}

	statusHaystack := renderForLeakCheck(t, f.storedDTPrometheus(t))
	for _, secret := range secrets {
		assert.NotContains(t, statusHaystack, secret, "the DTPrometheus leaks a token value in plain text")
	}
}

// renderForLeakCheck serializes an object to JSON so that data, env values, annotations, labels
// and status are all covered by one substring search. Secret data would be base64 here, but the
// controller creates no Secrets, so any occurrence of a token is a genuine plain-text leak.
func renderForLeakCheck(t *testing.T, obj client.Object) string {
	t.Helper()

	encoded, err := json.Marshal(obj)
	require.NoError(t, err)

	return string(encoded)
}

func (f *fixture) assertProxySecretIsReferenced(t *testing.T, clt client.Client) {
	t.Helper()

	const proxySecretName = "proxy-secret"

	integrationtests.CreateKubernetesObject(t, clt, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: proxySecretName, Namespace: f.ns},
		Data:       map[string][]byte{dynakube.ProxyKey: []byte("http://proxy.example.com:3128")},
	})
	f.dk.Spec.Proxy = &value.Source{ValueFrom: proxySecretName}
	require.NoError(t, clt.Update(t.Context(), f.dk))

	f.reconcileSuccessfully(t)

	container := containerByName(t, &f.gatewayStatefulSet(t).Spec.Template, "gateway")

	var found int

	for _, env := range container.Env {
		if env.Name != "HTTPS_PROXY" && env.Name != "HTTP_PROXY" {
			continue
		}

		found++

		assert.Emptyf(t, env.Value, "%s must not carry the proxy URL inline", env.Name)
		require.NotNil(t, env.ValueFrom)
		require.NotNil(t, env.ValueFrom.SecretKeyRef)
		assert.Equal(t, proxySecretName, env.ValueFrom.SecretKeyRef.Name)

		f.assertEnvSourceResolves(t, env)
	}

	assert.Equal(t, 2, found, "both HTTP_PROXY and HTTPS_PROXY must be set from the proxy Secret")
}
