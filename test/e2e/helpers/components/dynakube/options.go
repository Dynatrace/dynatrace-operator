//go:build e2e

package dynakube

import (
	"maps"
	"os"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/registry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type Option func(dk *dynakube.DynaKube)

func New(opts ...Option) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        defaultName,
			Namespace:   operator.DefaultNamespace,
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{},
	}
	for _, opt := range opts {
		opt(dk)
	}

	return dk
}

func WithName(name string) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Name = name
	}
}

func WithCustomCAs(configMapName string) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.TrustedCAs = configMapName
	}
}

func WithAnnotations(annotations map[string]string) Option {
	return func(dk *dynakube.DynaKube) {
		maps.Copy(dk.Annotations, annotations)
	}
}

func WithAPIURL(apiURL string) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.APIURL = apiURL
	}
}

func WithActiveGate() Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{
				activegate.KubeMonCapability.DisplayName,
				activegate.DynatraceAPICapability.DisplayName,
				activegate.RoutingCapability.DisplayName,
				activegate.MetricsIngestCapability.DisplayName,
				activegate.DebuggingCapability.DisplayName,
			},
		}
	}
}

func WithActiveGateModules(capabilities ...activegate.CapabilityDisplayName) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.ActiveGate = activegate.Spec{
			Capabilities: []activegate.CapabilityDisplayName{},
		}
		dk.Spec.ActiveGate.Capabilities = append(dk.Spec.ActiveGate.Capabilities, capabilities...)
	}
}

func WithActiveGateTLSSecret(tlsSecretName string) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.ActiveGate.TLSSecretName = tlsSecretName
	}
}

func WithCustomActiveGateImage(imageURI string) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.ActiveGate.Image = imageURI
	}
}

func WithActiveGateReplicas(replicas *int32) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.ActiveGate.Replicas = replicas
	}
}

func WithMetadataEnrichment() Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)
	}
}

func WithNameBasedOneAgentNamespaceSelector() Option {
	return func(dk *dynakube.DynaKube) {
		namespaceSelector := metav1.LabelSelector{
			MatchLabels: map[string]string{
				"oa-inject": dk.Name,
			},
		}
		switch {
		case dk.OneAgent().IsCloudNativeFullstackMode():
			dk.Spec.OneAgent.CloudNativeFullStack.NamespaceSelector = namespaceSelector
		case dk.OneAgent().IsApplicationMonitoringMode():
			dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = namespaceSelector
		}
	}
}

func WithNameBasedMetadataEnrichmentNamespaceSelector() Option {
	return func(dk *dynakube.DynaKube) {
		namespaceSelector := metav1.LabelSelector{
			MatchLabels: map[string]string{
				"me-inject": dk.Name,
			},
		}
		dk.Spec.MetadataEnrichment.NamespaceSelector = namespaceSelector
	}
}

func WithOneAgentNamespaceSelector(selector metav1.LabelSelector) Option {
	return func(dk *dynakube.DynaKube) {
		switch {
		case dk.OneAgent().IsCloudNativeFullstackMode():
			dk.Spec.OneAgent.CloudNativeFullStack.NamespaceSelector = selector
		case dk.OneAgent().IsApplicationMonitoringMode():
			dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = selector
		}
	}
}

func WithProxy(proxy *value.Source) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.Proxy = proxy
	}
}

func WithIstioIntegration() Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.EnableIstio = true
	}
}

func WithClassicFullstackSpec(classicFullStackSpec *oneagent.HostInjectSpec) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.OneAgent.ClassicFullStack = classicFullStackSpec
	}
}

func WithCloudNativeSpec(cloudNativeFullStackSpec *oneagent.CloudNativeFullStackSpec) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.OneAgent.CloudNativeFullStack = cloudNativeFullStackSpec
	}
}

func WithHostMonitoringSpec(hostInjectSpec *oneagent.HostInjectSpec) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.OneAgent.HostMonitoring = hostInjectSpec
	}
}

func WithApplicationMonitoringSpec(applicationMonitoringSpec *oneagent.ApplicationMonitoringSpec) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.OneAgent.ApplicationMonitoring = applicationMonitoringSpec
	}
}

func WithExtensionsPrometheusEnabledSpec(promEnabled bool) Option {
	return func(dk *dynakube.DynaKube) {
		if promEnabled {
			dk.Spec.Extensions = &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}}
			dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = true
		} else {
			dk.Spec.Extensions = nil
		}
	}
}

func WithExtensionsEECImageRef(t *testing.T) Option {
	return func(dk *dynakube.DynaKube) {
		if setImageRefFromEnvs(
			t,
			dk,
			&dk.Spec.Templates.ExtensionExecutionController.ImageRef,
			consts.EECImageEnvVar,
			consts.DefaultEECRepo,
		) {
			// Disable legacy mounts when using a non-default image
			dk.Annotations["feature.dynatrace.com/use-eec-legacy-mounts"] = "false"
		}
	}
}

func WithLogMonitoring() Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.LogMonitoring = &logmonitoring.Spec{}
	}
}

func WithLogMonitoringImageRef(t *testing.T) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{}
		setImageRefFromEnvs(
			t,
			dk,
			&dk.Spec.Templates.LogMonitoring.ImageRef,
			consts.LogMonitoringImageEnvVar,
			consts.DefaultLogMonitoringRepo,
		)
	}
}

func WithKSPM() Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.KSPM = &kspm.Spec{}
	}
}

func WithKSPMImageRef(t *testing.T) Option {
	return func(dk *dynakube.DynaKube) {
		setImageRefFromEnvs(
			t,
			dk,
			&dk.Spec.Templates.KSPMNodeConfigurationCollector.ImageRef,
			consts.KSPMImageEnvVar,
			consts.DefaultKSPMRepo,
		)
	}
}

func WithTelemetryIngestEnabled(enabled bool, protocols ...otelcgen.Protocol) Option {
	return func(dk *dynakube.DynaKube) {
		if enabled {
			dk.Spec.TelemetryIngest = &telemetryingest.Spec{}
			dk.Spec.TelemetryIngest.Protocols = append(dk.Spec.TelemetryIngest.Protocols, protocols...)
		} else {
			dk.Spec.TelemetryIngest = nil
		}
	}
}

func WithTelemetryIngestEndpointTLS(secretName string) Option {
	return func(dk *dynakube.DynaKube) {
		if dk.Spec.TelemetryIngest == nil {
			dk.Spec.TelemetryIngest = &telemetryingest.Spec{}
		}
		dk.Spec.TelemetryIngest.TLSRefName = secretName
	}
}

func WithOTelCollectorImageRef(t *testing.T) Option {
	return func(dk *dynakube.DynaKube) {
		setImageRefFromEnvs(
			t,
			dk,
			&dk.Spec.Templates.OpenTelemetryCollector.ImageRef,
			consts.OtelCollectorImageEnvVar,
			consts.DefaultOtelCollectorRepo,
		)
	}
}

func WithOTelCollectorReplicas(replicas *int32) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.Templates.OpenTelemetryCollector.Replicas = replicas
	}
}

func WithExtensionsDatabases(databases ...extensions.DatabaseSpec) Option {
	return func(dk *dynakube.DynaKube) {
		if dk.Spec.Extensions == nil {
			dk.Spec.Extensions = &extensions.Spec{}
		}
		dk.Spec.Extensions.Databases = databases
	}
}

func WithExtensionsDBExecutorImageRef(t *testing.T) Option {
	return func(dk *dynakube.DynaKube) {
		setImageRefFromEnvs(
			t,
			dk,
			&dk.Spec.Templates.SQLExtensionExecutor.ImageRef,
			consts.DBExecutorImageEnvVar,
			consts.DefaultDBExecutorRepo,
		)
	}
}

func setImageRefFromEnvs(t *testing.T, dk *dynakube.DynaKube, imageRef *image.Ref, envVar, defaultRepo string) bool {
	t.Helper()

	envImage := os.Getenv(envVar)
	if envImage != "" {
		repo, tag, _ := strings.Cut(envImage, ":")
		imageRef.Repository = repo
		imageRef.Tag = tag

		if repo != defaultRepo {
			dk.Spec.CustomPullSecret = consts.DevRegistryPullSecretName
			t.Logf("using private repo image from env %s: %s", envVar, envImage)

			return true
		}

		t.Logf("using image from env %s: %s", envVar, envImage)

		return false
	}

	uri := registry.GetLatestImageURI(t, defaultRepo)
	imageRef.Repository, imageRef.Tag, _ = strings.Cut(uri, ":")

	return false
}
