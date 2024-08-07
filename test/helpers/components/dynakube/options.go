//go:build e2e

package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		for key, value := range annotations {
			dk.ObjectMeta.Annotations[key] = value
		}
	}
}

func WithApiUrl(apiUrl string) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.APIURL = apiUrl
	}
}

func WithActiveGate() Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.ActiveGate = dynakube.ActiveGateSpec{
			Capabilities: []dynakube.CapabilityDisplayName{
				dynakube.KubeMonCapability.DisplayName,
				dynakube.DynatraceApiCapability.DisplayName,
				dynakube.RoutingCapability.DisplayName,
				dynakube.MetricsIngestCapability.DisplayName,
			},
		}
	}
}

func WithMetadataEnrichment() Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.MetadataEnrichment.Enabled = true
	}
}

func WithActiveGateTlsSecret(tlsSecretName string) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.ActiveGate.TlsSecretName = tlsSecretName
	}
}

func WithCustomActiveGateImage(imageURI string) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.ActiveGate.Image = imageURI
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
		case dk.CloudNativeFullstackMode():
			dk.Spec.OneAgent.CloudNativeFullStack.NamespaceSelector = namespaceSelector
		case dk.ApplicationMonitoringMode():
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
		case dk.CloudNativeFullstackMode():
			dk.Spec.OneAgent.CloudNativeFullStack.NamespaceSelector = selector
		case dk.ApplicationMonitoringMode():
			dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = selector
		}
	}
}

func WithProxy(proxy *dynakube.DynaKubeProxy) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.Proxy = proxy
	}
}

func WithIstioIntegration() Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.EnableIstio = true
	}
}

func WithClassicFullstackSpec(classicFullStackSpec *dynakube.HostInjectSpec) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.OneAgent.ClassicFullStack = classicFullStackSpec
	}
}

func WithCloudNativeSpec(cloudNativeFullStackSpec *dynakube.CloudNativeFullStackSpec) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.OneAgent.CloudNativeFullStack = cloudNativeFullStackSpec
	}
}

func WithApplicationMonitoringSpec(applicationMonitoringSpec *dynakube.ApplicationMonitoringSpec) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.OneAgent.ApplicationMonitoring = applicationMonitoringSpec
	}
}

func WithNetworkZone(networkZone string) Option {
	return func(dk *dynakube.DynaKube) {
		dk.Spec.NetworkZone = networkZone
	}
}
