//go:build e2e

package dynakube

import (
	dynakubev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Option func(dynakube *dynakubev1beta1.DynaKube)

func New(opts ...Option) *dynakubev1beta1.DynaKube {
	dynakube := &dynakubev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        defaultName,
			Namespace:   operator.DefaultNamespace,
			Annotations: map[string]string{},
		},
		Spec: dynakubev1beta1.DynaKubeSpec{},
	}
	for _, opt := range opts {
		opt(dynakube)
	}

	return dynakube
}

func WithName(name string) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Name = name
	}
}

func WithCustomPullSecret(secretName string) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.CustomPullSecret = secretName
	}
}

func WithCustomCAs(configMapName string) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.TrustedCAs = configMapName
	}
}

func WithAnnotations(annotations map[string]string) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		for key, value := range annotations {
			dynakube.ObjectMeta.Annotations[key] = value
		}
	}
}

func WithApiUrl(apiUrl string) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.APIURL = apiUrl
	}
}

func WithActiveGate() Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.ActiveGate = dynakubev1beta1.ActiveGateSpec{
			Capabilities: []dynakubev1beta1.CapabilityDisplayName{
				dynakubev1beta1.KubeMonCapability.DisplayName,
				dynakubev1beta1.DynatraceApiCapability.DisplayName,
				dynakubev1beta1.RoutingCapability.DisplayName,
				dynakubev1beta1.MetricsIngestCapability.DisplayName,
			},
		}
	}
}

func WithCustomActiveGateImage(imageURI string) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.ActiveGate.Image = imageURI
	}
}

func WithNameBasedNamespaceSelector() Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.NamespaceSelector = metav1.LabelSelector{
			MatchLabels: map[string]string{
				"inject": dynakube.Name,
			},
		}
	}
}

func WithNamespaceSelector(selector metav1.LabelSelector) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.NamespaceSelector = selector
	}
}

func WithProxy(proxy *dynakubev1beta1.DynaKubeProxy) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.Proxy = proxy
	}
}

func WithIstioIntegration() Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.EnableIstio = true
	}
}

func WithClassicFullstackSpec(classicFullStackSpec *dynakubev1beta1.HostInjectSpec) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.OneAgent.ClassicFullStack = classicFullStackSpec
	}
}

func WithCloudNativeSpec(cloudNativeFullStackSpec *dynakubev1beta1.CloudNativeFullStackSpec) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.OneAgent.CloudNativeFullStack = cloudNativeFullStackSpec
	}
}

func WithApplicationMonitoringSpec(applicationMonitoringSpec *dynakubev1beta1.ApplicationMonitoringSpec) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.OneAgent.ApplicationMonitoring = applicationMonitoringSpec
	}
}

func WithNetworkZone(networkZone string) Option {
	return func(dynakube *dynakubev1beta1.DynaKube) {
		dynakube.Spec.NetworkZone = networkZone
	}
}
