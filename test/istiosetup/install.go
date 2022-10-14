package istiosetup

import (
	"context"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	istioNamespace         = "istio-system"
	istioInitContainerName = "istio-init"
)

var IstioLabel = map[string]string{
	"istio-injection": "enabled",
}

func AssertIstioNamespace() func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		var namespace corev1.Namespace
		err := environmentConfig.Client().Resources().Get(ctx, istioNamespace, "", &namespace)
		return ctx, errors.WithStack(err)
	}
}

func AssertIstiodDeployment() func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		var deployment appsv1.Deployment
		err := environmentConfig.Client().Resources().Get(ctx, "istiod", "istio-system", &deployment)
		return ctx, errors.WithStack(err)
	}
}

func AssessIstio(builder *features.FeatureBuilder) {
	builder.Assess("sample apps have working istio init container", checkSampleAppIstioInitContainers)
	builder.Assess("operator pods have working istio init container", checkOperatorIstioInitContainers)
	builder.Assess("istio virtual service for ApiUrl created", checkVirtualServiceForApiUrl)
	builder.Assess("istio service entry for ApiUrl created", checkServiceEntryForApiUrl)
}

func checkSampleAppIstioInitContainers(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()
	pods := sampleapps.Get(t, ctx, resources)

	assertIstioInitContainer(t, pods)
	return ctx
}

func checkOperatorIstioInitContainers(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()
	var pods corev1.PodList
	require.NoError(t, resources.WithNamespace(dynakube.Namespace).List(ctx, &pods))

	assertIstioInitContainer(t, pods)
	return ctx
}

func assertIstioInitContainer(t *testing.T, pods corev1.PodList) {
	for _, podItem := range pods.Items {
		if podItem.DeletionTimestamp != nil {
			continue
		}

		require.NotNil(t, podItem)
		require.NotNil(t, podItem.Spec)

		if strings.HasPrefix(podItem.Name, "dynakube-oneagent") {
			continue
		}

		require.NotEmpty(t, podItem.Spec.InitContainers, "'%s' pod has no init containers", podItem.Name)

		istioInitFound := false

		for _, initContainer := range podItem.Spec.InitContainers {
			if initContainer.Name == istioInitContainerName {
				istioInitFound = true
				break
			}
		}
		assert.True(t, istioInitFound, "'%s' pod - '%s' init container not found", podItem.Name, istioInitContainerName)
	}
}

func checkVirtualServiceForApiUrl(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	apiHost := apiUrlCommunicationHost(t)
	serviceName := istio.BuildNameForEndpoint(dynakube.Name, apiHost.Protocol, apiHost.Host, apiHost.Port)

	vs, err := istioClient(t, environmentConfig.Client().RESTConfig()).NetworkingV1alpha3().VirtualServices(dynakube.Namespace).Get(ctx, serviceName, v1.GetOptions{})
	require.Nil(t, err, "istio: faild to get '%s' virtual service object", serviceName)

	require.NotEmpty(t, vs.ObjectMeta.OwnerReferences)
	assert.Equal(t, dynakube.Name, vs.ObjectMeta.OwnerReferences[0].Name)

	require.NotEmpty(t, vs.Spec.Hosts)
	assert.Equal(t, apiHost.Host, vs.Spec.Hosts[0])

	return ctx
}

func checkServiceEntryForApiUrl(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	apiHost := apiUrlCommunicationHost(t)
	serviceName := istio.BuildNameForEndpoint(dynakube.Name, apiHost.Protocol, apiHost.Host, apiHost.Port)

	vs, err := istioClient(t, environmentConfig.Client().RESTConfig()).NetworkingV1alpha3().ServiceEntries(dynakube.Namespace).Get(ctx, serviceName, v1.GetOptions{})
	require.Nil(t, err, "istio: failed to get '%s' service entry object", serviceName)

	require.NotEmpty(t, vs.ObjectMeta.OwnerReferences)
	assert.Equal(t, dynakube.Name, vs.ObjectMeta.OwnerReferences[0].Name)

	require.NotEmpty(t, vs.Spec.Hosts)
	assert.Equal(t, apiHost.Host, vs.Spec.Hosts[0])

	return ctx
}

func istioClient(t *testing.T, restConfig *rest.Config) *istioclientset.Clientset {
	client, err := istioclientset.NewForConfig(restConfig)
	require.Nil(t, err, "istio: failed to initialize client")
	return client
}

func apiUrlCommunicationHost(t *testing.T) dtclient.CommunicationHost {
	secretConfig, err := secrets.DefaultSingleTenant(afero.NewOsFs())
	require.NoError(t, err)

	apiHost, err := dtclient.ParseEndpoint(secretConfig.ApiUrl)
	require.Nil(t, err)

	return apiHost
}
