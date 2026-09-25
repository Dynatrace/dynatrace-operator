// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

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
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
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
	CollectorNamespace string                `json:"collector_namespace"`
	CollectorSelector  *metav1.LabelSelector `json:"collector_selector"`
	PrometheusCR       struct {
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
		Endpoint string `json:"endpoint"`
		Interval string `json:"interval"`
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

	return unmarshalConfig[collectorConfig](t, f.getConfigMap(t, f.pm.Gateway().GetStatefulSetName()), "relay")
}

func (f *fixture) scraperConfig(t *testing.T) collectorConfig {
	t.Helper()

	return unmarshalConfig[collectorConfig](t, f.getConfigMap(t, f.pm.Scraper().GetDeploymentName()), "scraper")
}

func (f *fixture) targetAllocatorConfig(t *testing.T) targetAllocatorConfig {
	t.Helper()

	return unmarshalConfig[targetAllocatorConfig](t,
		f.getConfigMap(t, f.pm.TargetAllocator().GetDeploymentName()), "targetallocator.yaml")
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

// testConfigMaps checks what the golden files in the component packages cannot: that each rendered
// config is internally consistent, and that the values the apiserver defaulted into the CRD arrive
// in it. The exact rendered bytes are pinned by those golden files, from an explicitly populated
// spec, so nothing here restates a field they already cover.
func testConfigMaps(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "configmaps")
	f.assertReconcileSuccessfully(t)

	t.Run("gateway collector config", func(t *testing.T) {
		cfg := f.gatewayConfig(t)
		assertCollectorConfigConsistent(t, cfg)

		// The transform processor tags every datapoint with the cluster name through a placeholder
		// the container has to define, which no single-object golden file can check.
		require.Contains(t, string(cfg.Processors["transform"]), "${env:K8S_CLUSTER_NAME}",
			"the gateway config no longer reads the cluster name from the environment")

		gatewayContainer := containerByName(t, gatewayWorkload(t, f).template, "gateway")
		clusterName := envByName(t, gatewayContainer, "K8S_CLUSTER_NAME")
		assert.Equal(t, testClusterName, clusterName.Value, "the cluster name must come from the DynaKube status")
	})

	t.Run("scraper collector config", func(t *testing.T) {
		cfg := f.scraperConfig(t)
		assertCollectorConfigConsistent(t, cfg)

		receiver := unmarshalInto[scraperReceiver](t, cfg.Receivers["prometheus"], "scraper prometheus receiver")
		assert.Equal(t, "1m0s", receiver.TargetAllocator.Interval, "the CRD default for targetsPollInterval is 60s")

		// The config expands a placeholder the container has to define, which no single-object
		// golden file can check.
		envByName(t, containerByName(t, scraperWorkload(t, f).template, "scraper"), "MY_POD_NAME")
	})

	t.Run("target allocator config carries the CRD defaults", func(t *testing.T) {
		// The golden file renders an explicitly populated spec, so the defaults the apiserver
		// applies when the user writes nothing are only observable here.
		cfg := f.targetAllocatorConfig(t)

		assert.Equal(t, "1m0s", cfg.PrometheusCR.ScrapeInterval, "the CRD default for scrapeInterval is 60s")

		defaultSelector := &metav1.LabelSelector{
			MatchLabels: map[string]string{prometheusmonitoring.DefaultCustomResourceSelectorLabel: "true"},
		}
		assert.Equal(t, defaultSelector, cfg.PrometheusCR.PodMonitorSelector)
		assert.Equal(t, defaultSelector, cfg.PrometheusCR.ServiceMonitorSelector)
		assert.Equal(t, defaultSelector, cfg.PrometheusCR.ScrapeConfigSelector)
		assert.Equal(t, defaultSelector, cfg.PrometheusCR.ProbeSelector)

		assert.Nil(t, cfg.PrometheusCR.PodMonitorNamespaceSelector, "all namespaces by default")
		assert.Nil(t, cfg.PrometheusCR.ServiceMonitorNamespaceSelector)
		assert.Nil(t, cfg.PrometheusCR.ScrapeConfigNamespaceSelector)
		assert.Nil(t, cfg.PrometheusCR.ProbeNamespaceSelector)
	})

	// A custom customResourceSelector is not covered here: the target allocator's golden ConfigMap
	// is rendered from a spec that sets both the selector and the namespace selector, so it already
	// pins all eight fields the fan-out produces.

	t.Run("a custom CA on the DynaKube reaches the gateway exporter and the pod", func(t *testing.T) {
		integrationtests.CreateKubernetesObject(t, clt, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "custom-cas", Namespace: f.ns},
			Data:       map[string]string{"certs": "-----BEGIN CERTIFICATE-----"},
		})
		f.dk.Spec.TrustedCAs = "custom-cas"
		require.NoError(t, clt.Update(t.Context(), f.dk))
		f.assertReconcileSuccessfully(t)

		exporter := unmarshalInto[otlpHTTPExporter](t, f.gatewayConfig(t).Exporters["otlp_http"], "gateway exporter")
		require.NotEmpty(t, exporter.TLS.CAFile)

		// The path in the config must be a path the pod actually has mounted.
		sts := f.getGatewayStatefulSet(t)
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

// testWiring is the cross-component test: it verifies that the names and ports the configs point
// at match the Service and container objects that actually exist, so the whole path
// scraper -> target allocator and scraper -> gateway -> Dynatrace is consistent.
func testWiring(t *testing.T, clt client.Client) {
	f := newFixture(t, clt, "wiring")
	f.assertReconcileSuccessfully(t)

	t.Run("scraper reaches the target allocator", func(t *testing.T) {
		receiver := unmarshalInto[scraperReceiver](t, f.scraperConfig(t).Receivers["prometheus"], "prometheus receiver")

		host, port := splitEndpoint(t, receiver.TargetAllocator.Endpoint)
		svc := f.getService(t, f.pm.TargetAllocator().GetDeploymentName())

		assert.Equal(t, svc.Name+"."+svc.Namespace, host,
			"the configured target allocator host must be the target allocator Service")

		svcPort := servicePortByNumber(t, svc, port)
		assert.Equal(t, "http-port", svcPort.Name, "the scraper talks plain HTTP to the target allocator")

		// ... and that Service port must land on a port the allocator container actually opens,
		// on the address the allocator is configured to listen on.
		taTemplate := targetAllocatorWorkload(t, f).template
		container := containerByName(t, taTemplate, "targetallocator")
		containerPort := containerPortByName(t, container, svcPort.TargetPort.StrVal)

		listenPort := listenAddrPort(t, f.targetAllocatorConfig(t).ListenAddr)
		assert.Equal(t, listenPort, containerPort.ContainerPort,
			"the allocator listens on a different port than the one the Service targets")
	})

	t.Run("target allocator finds the scraper pods", func(t *testing.T) {
		cfg := f.targetAllocatorConfig(t)
		scraperDeploy := f.getDeployment(t, f.pm.Scraper().GetDeploymentName())

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

		svc := f.getService(t, f.pm.Gateway().GetStatefulSetName())
		assert.Equal(t, svc.Name+"."+svc.Namespace, exporter.Resolver.K8s.Service,
			"the load balancing resolver must point at the gateway Service")
		require.Len(t, exporter.Resolver.K8s.Ports, 1)

		svcPort := servicePortByNumber(t, svc, exporter.Resolver.K8s.Ports[0])
		assert.Equal(t, "otlp", svcPort.Name)

		gwTemplate := gatewayWorkload(t, f).template
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
		gwTemplate := gatewayWorkload(t, f).template
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

// testSecrets covers the only Secret in play: the DynaKube token Secret, which the controller reads
// but never copies. Each subtest gets its own fixture, because they mutate the DynaKube and create
// Secrets whose t.Cleanup would otherwise pull the ground out from under the next subtest.
func testSecrets(t *testing.T, clt client.Client) {
	t.Run("no token value is ever written in plain text", func(t *testing.T) {
		f := newFixture(t, clt, "secrets-leak")
		f.assertReconcileSuccessfully(t)

		f.assertNoTokenLeak(t)
	})

	t.Run("rotating the token does not roll any pod", func(t *testing.T) {
		// The token is mounted as a projected directory (not a subPath) precisely so the kubelet
		// refreshes it in place and the bearertokenauth extension picks the new value up. Nothing
		// derives from its content, so no hash may change and no pod may restart.
		f := newFixture(t, clt, "secrets-rotation")
		f.assertReconcileSuccessfully(t)

		before := f.resourceVersions(t)
		beforeHashes := f.configHashes(t)

		secret := f.getTokenSecret(t)
		secret.Data[token.DataIngestKey] = []byte("dt0c01.ROTATEDDATAINGESTTOKENVALUE")
		require.NoError(t, clt.Update(t.Context(), secret))

		f.assertReconcileSuccessfully(t)

		assert.Equal(t, before, f.resourceVersions(t))
		assert.Equal(t, beforeHashes, f.configHashes(t))
		f.assertNoTokenLeak(t)
	})

	t.Run("a renamed token secret is followed", func(t *testing.T) {
		f := newFixture(t, clt, "secrets-rename")
		f.assertReconcileSuccessfully(t)

		integrationtests.CreateKubernetesObject(t, clt, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "other-tokens", Namespace: f.ns},
			Data: map[string][]byte{
				token.APIKey:        []byte(testAPIToken),
				token.DataIngestKey: []byte(testDataIngestToken),
			},
		})
		f.dk.Spec.Tokens = "other-tokens"
		require.NoError(t, clt.Update(t.Context(), f.dk))

		f.assertReconcileSuccessfully(t)

		sts := f.getGatewayStatefulSet(t)
		for _, volume := range sts.Spec.Template.Spec.Volumes {
			f.assertVolumeSourceResolves(t, volume)
		}

		assert.Equal(t, "other-tokens", projectedSecretName(t, sts.Spec.Template.Spec.Volumes, "dt-token"))
	})

	t.Run("proxy credentials are referenced, never inlined", func(t *testing.T) {
		f := newFixture(t, clt, "secrets-proxy")
		f.assertReconcileSuccessfully(t)

		f.assertProxySecretIsReferenced(t, clt)
	})

	t.Run("a custom pull secret reaches every component", func(t *testing.T) {
		// Each component has its own unit test for the reference. What only one full reconcile shows
		// is that all three get it from the same DynaKube field: a component that never wired it up
		// leaves exactly one pod template unable to pull, which is invisible per component.
		f := newFixture(t, clt, "secrets-pull")

		const pullSecretName = "custom-pull-secret"

		integrationtests.CreateKubernetesObject(t, clt, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: pullSecretName, Namespace: f.ns},
			Type:       corev1.SecretTypeDockerConfigJson,
			Data:       map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{}}`)},
		})
		f.dk.Spec.CustomPullSecret = pullSecretName
		require.NoError(t, clt.Update(t.Context(), f.dk))

		f.assertReconcileSuccessfully(t)

		for _, c := range components() {
			tmpl := c.getWorkload(t, f).template

			assert.Equalf(t, []corev1.LocalObjectReference{{Name: pullSecretName}}, tmpl.Spec.ImagePullSecrets,
				"%s does not reference the DynaKube's custom pull secret", c.name)

			for _, pullSecret := range tmpl.Spec.ImagePullSecrets {
				f.assertSecretExists(t, pullSecret.Name)
			}
		}
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

// assertNoTokenLeak walks everything the operator wrote into the namespace, plus the PrometheusMonitoring
// status, and fails if a token value shows up in plain text anywhere. Secret values may only be
// consumed through a secretKeyRef or a Secret volume.
func (f *fixture) assertNoTokenLeak(t *testing.T) {
	t.Helper()

	secrets := []string{testAPIToken, testDataIngestToken}

	for _, want := range expectedManagedObjects(f.pm) {
		obj := f.getManagedObject(t, want)
		haystack := renderForLeakCheck(t, obj)

		for _, secret := range secrets {
			assert.NotContainsf(t, haystack, secret, "%s leaks a token value in plain text", want)
		}
	}

	statusHaystack := renderForLeakCheck(t, f.getStoredPrometheusMonitoring(t))
	for _, secret := range secrets {
		assert.NotContains(t, statusHaystack, secret, "the PrometheusMonitoring leaks a token value in plain text")
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

	f.assertReconcileSuccessfully(t)

	container := containerByName(t, &f.getGatewayStatefulSet(t).Spec.Template, "gateway")

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

	// Whatever the proxy is, in-cluster traffic must not go through it: the gateway talks to the
	// apiserver for the k8s_attributes processor, and the scraper reaches the gateway by Service
	// DNS. A NO_PROXY that misses those turns a working proxy setup into no telemetry at all.
	noProxy := envByName(t, container, "NO_PROXY")
	assert.Contains(t, noProxy.Value, "$(KUBERNETES_SERVICE_HOST)")
	assert.Contains(t, noProxy.Value, "kubernetes.default")
}

func envByName(t *testing.T, container corev1.Container, name string) corev1.EnvVar {
	t.Helper()

	for _, env := range container.Env {
		if env.Name == name {
			return env
		}
	}

	require.Failf(t, "env not found", "container %s does not define %s", container.Name, name)

	return corev1.EnvVar{}
}
