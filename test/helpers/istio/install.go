//go:build e2e

package istio

import (
	"context"
	"os"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	istioNamespace                  = "istio-system"
	istioInitContainerName          = "istio-init"
	openshiftIstioInitContainerName = "istio-validation"
	enforceIstioEnv                 = "ENFORCE_ISTIO"
)

func enforceIstio() bool {
	return os.Getenv(enforceIstioEnv) == "true"
}

func AssertIstioNamespace() func(ctx context.Context, envConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, envConfig *envconf.Config, t *testing.T) (context.Context, error) {
		var namespace corev1.Namespace
		err := envConfig.Client().Resources().Get(ctx, istioNamespace, "", &namespace)
		if err != nil && !enforceIstio() {
			t.Skip("skipping istio test, istio namespace is not present")

			return ctx, nil
		}

		return ctx, errors.WithStack(err)
	}
}

func AssertIstiodDeployment() func(ctx context.Context, envConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, envConfig *envconf.Config, t *testing.T) (context.Context, error) {
		var deployment appsv1.Deployment
		err := envConfig.Client().Resources().Get(ctx, "istiod", "istio-system", &deployment)
		if err != nil && !enforceIstio() {
			t.Skip("skipping istio test, istiod deployment is not present")

			return ctx, nil
		}

		return ctx, errors.WithStack(err)
	}
}

func AssessIstio(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube, sampleApp sample.App) {
	builder.Assess("sample apps have working istio init container", checkSampleAppIstioInitContainers(sampleApp, testDynakube))
	builder.Assess("operator pods have working istio init container", checkOperatorIstioInitContainers(testDynakube))
	builder.Assess("istio virtual service for ApiUrl created", checkVirtualServiceForApiUrl(testDynakube))
	builder.Assess("istio service entry for ApiUrl created", checkServiceEntryForApiUrl(testDynakube))
}

func checkSampleAppIstioInitContainers(sampleApp sample.App, testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, resources)
		assertIstioInitContainer(t, pods, testDynakube)

		return ctx
	}
}

func checkOperatorIstioInitContainers(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		var pods corev1.PodList
		require.NoError(t, resources.WithNamespace(testDynakube.Namespace).List(ctx, &pods))

		assertIstioInitContainer(t, pods, testDynakube)

		return ctx
	}
}

func assertIstioInitContainer(t *testing.T, pods corev1.PodList, testDynakube dynatracev1beta1.DynaKube) {
	istioInitName := determineIstioInitContainerName(t)

	for _, podItem := range pods.Items {
		if podItem.DeletionTimestamp != nil {
			continue
		}

		require.NotNil(t, podItem)
		require.NotNil(t, podItem.Spec)

		if strings.HasPrefix(podItem.Name, testDynakube.OneAgentDaemonsetName()) {
			continue
		}

		require.NotEmpty(t, podItem.Spec.InitContainers, "'%s' pod has no init containers", podItem.Name)

		istioInitFound := false

		for _, initContainer := range podItem.Spec.InitContainers {
			if initContainer.Name == istioInitName {
				istioInitFound = true

				break
			}
		}
		assert.True(t, istioInitFound, "'%s' pod - '%s' init container not found", podItem.Name, istioInitName)
	}
}

func determineIstioInitContainerName(t *testing.T) string {
	istioInitName := istioInitContainerName
	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err)
	if isOpenshift {
		istioInitName = openshiftIstioInitContainerName
	}

	return istioInitName
}

func checkVirtualServiceForApiUrl(dynakube dynatracev1beta1.DynaKube) features.Func { //nolint: dupl
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		apiHost := apiUrlCommunicationHost(t, dynakube)
		serviceName := istio.BuildNameForFQDNServiceEntry(dynakube.Name, istio.OperatorComponent)

		virtualService, err := istioClient(t, envConfig.Client().RESTConfig()).NetworkingV1beta1().VirtualServices(dynakube.Namespace).Get(ctx, serviceName, metav1.GetOptions{})
		require.NoError(t, err, "istio: failed to get '%s' virtual service object", serviceName)

		require.NotEmpty(t, virtualService.ObjectMeta.OwnerReferences)
		assert.Equal(t, dynakube.Name, virtualService.ObjectMeta.OwnerReferences[0].Name)

		require.NotEmpty(t, virtualService.Spec.GetHosts())
		assert.Equal(t, apiHost.Host, virtualService.Spec.GetHosts()[0])

		return ctx
	}
}

func checkServiceEntryForApiUrl(dynakube dynatracev1beta1.DynaKube) features.Func { //nolint: dupl
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		apiHost := apiUrlCommunicationHost(t, dynakube)
		serviceName := istio.BuildNameForFQDNServiceEntry(dynakube.Name, istio.OperatorComponent)

		serviceEntry, err := istioClient(t, envConfig.Client().RESTConfig()).NetworkingV1beta1().ServiceEntries(dynakube.Namespace).Get(ctx, serviceName, metav1.GetOptions{})
		require.NoError(t, err, "istio: failed to get '%s' service entry object", serviceName)

		require.NotEmpty(t, serviceEntry.ObjectMeta.OwnerReferences)
		assert.Equal(t, dynakube.Name, serviceEntry.ObjectMeta.OwnerReferences[0].Name)

		require.NotEmpty(t, serviceEntry.Spec.GetHosts())
		assert.Equal(t, apiHost.Host, serviceEntry.Spec.GetHosts()[0])

		return ctx
	}
}

func istioClient(t *testing.T, restConfig *rest.Config) *istioclientset.Clientset {
	client, err := istioclientset.NewForConfig(restConfig)
	require.NoError(t, err, "istio: failed to initialize client")

	return client
}

func apiUrlCommunicationHost(t *testing.T, dynakube dynatracev1beta1.DynaKube) dtclient.CommunicationHost {
	apiHost, err := dtclient.ParseEndpoint(dynakube.ApiUrl())
	require.NoError(t, err)

	return apiHost
}
