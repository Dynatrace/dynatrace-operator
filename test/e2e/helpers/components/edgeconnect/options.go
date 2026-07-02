//go:build e2e

package edgeconnect

import (
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/registry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultName      = "edgeconnect"
	defaultNamespace = "dynatrace"

	defaultECRepo = "public.ecr.aws/dynatrace/edgeconnect"
	ecImageEnvVar = "E2E_EDGECONNECT_IMAGE"
)

type Option func(ec *edgeconnect.EdgeConnect)

func New(opts ...Option) *edgeconnect.EdgeConnect {
	ec := &edgeconnect.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultName,
			Namespace: defaultNamespace,
		},
		Spec:   edgeconnect.EdgeConnectSpec{},
		Status: edgeconnect.EdgeConnectStatus{},
	}
	for _, opt := range opts {
		opt(ec)
	}

	return ec
}

func WithName(name string) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Name = name
	}
}

func WithAPIServer(apiURL string) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Spec.APIServer = apiURL
	}
}

func WithOAuthClientSecret(clientSecretName string) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Spec.OAuth.ClientSecret = clientSecretName
	}
}

func WithOAuthResource(resource string) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Spec.OAuth.Resource = resource
	}
}

func WithOAuthEndpoint(endpoint string) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Spec.OAuth.Endpoint = endpoint
	}
}

func WithProvisionerMode(enabled bool) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Spec.OAuth.Provisioner = enabled
	}
}

func WithK8SAutomationMode(enabled bool) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Spec.KubernetesAutomation = &edgeconnect.KubernetesAutomationSpec{
			Enabled: enabled,
		}
	}
}

func WithHostPattern(hostPattern string) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		if ec.Spec.HostPatterns == nil {
			ec.Spec.HostPatterns = make([]string, 0)
		}
		ec.Spec.HostPatterns = append(ec.Spec.HostPatterns, hostPattern)
	}
}

func WithServiceAccount(serviceAccountName string) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Spec.ServiceAccountName = &serviceAccountName
	}
}

func WithEnvValue(key, value string) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Spec.Env = append(ec.Spec.Env, corev1.EnvVar{Name: key, Value: value})
	}
}

func WithCACert(refName string) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Spec.CaCertsRef = refName
	}
}

func WithProxy(spec *proxy.Spec) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Spec.Proxy = spec
	}
}

func WithReplicas(replicas *int32) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		ec.Spec.Replicas = replicas
	}
}

func GetLatestImageTagURI(t *testing.T) string {
	t.Helper()

	return registry.GetLatestImageURI(t, defaultECRepo, ecImageEnvVar, false)
}

func GetLatestImageDigestURI(t *testing.T) string {
	t.Helper()

	return registry.GetLatestImageURI(t, defaultECRepo, ecImageEnvVar, true)
}

func WithImageRef(t *testing.T, imageURI string) Option {
	return func(ec *edgeconnect.EdgeConnect) {
		if strings.Contains(imageURI, "@") {
			ec.Spec.ImageRef.Repository, ec.Spec.ImageRef.Digest, _ = strings.Cut(imageURI, "@")
		} else {
			ec.Spec.ImageRef.Repository, ec.Spec.ImageRef.Tag, _ = strings.Cut(imageURI, ":")
		}
		setCustomPullSecretIfNeeded(t, ec)
	}
}

func setCustomPullSecretIfNeeded(t *testing.T, ec *edgeconnect.EdgeConnect) {
	t.Helper()

	if ec.Spec.ImageRef.Repository != defaultECRepo {
		ec.Spec.CustomPullSecret = consts.DevRegistryPullSecretName
		t.Logf("image repo %s differs from default %s, setting custom pull secret", ec.Spec.ImageRef.Repository, defaultECRepo)
	}
}
