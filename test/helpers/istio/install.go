//go:build e2e

package istio

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
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

func AssessIstio(builder *features.FeatureBuilder, testDynakube dynakube.DynaKube, sampleApp sample.App) {
	builder.Assess("sample apps have working istio init container", checkSampleAppIstioInitContainers(sampleApp, testDynakube))
	builder.Assess("operator pods have working istio init container", checkOperatorIstioInitContainers(testDynakube))
	builder.Assess("istio virtual service for APIURL created", checkVirtualServiceForAPIURL(testDynakube))
	builder.Assess("istio service entry for APIURL created", checkServiceEntryForAPIURL(testDynakube))
}

func checkSampleAppIstioInitContainers(sampleApp sample.App, testDynakube dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, resources)
		assertIstioInitContainer(t, pods, testDynakube)

		return ctx
	}
}

func checkOperatorIstioInitContainers(testDynakube dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		var pods corev1.PodList
		require.NoError(t, resources.WithNamespace(testDynakube.Namespace).List(ctx, &pods))

		assertIstioInitContainer(t, pods, testDynakube)

		return ctx
	}
}

func assertIstioInitContainer(t *testing.T, pods corev1.PodList, testDynakube dynakube.DynaKube) {
	istioInitName := determineIstioInitContainerName(t)

	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}

		if strings.HasPrefix(pod.Name, testDynakube.OneAgent().GetDaemonsetName()) {
			continue
		}

		require.NotEmpty(t, pod.Spec.InitContainers, "'%s' pod has no init containers", pod.Name)

		istioInitFound := false

		for _, initContainer := range pod.Spec.InitContainers {
			if initContainer.Name == istioInitName {
				istioInitFound = true

				break
			}
		}
		assert.True(t, istioInitFound, "'%s' pod - '%s' init container not found", pod.Name, istioInitName)
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

func checkVirtualServiceForAPIURL(dk dynakube.DynaKube) features.Func { //nolint:dupl
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		apiHost := apiURLCommunicationHost(t, dk)
		serviceName := istio.BuildNameForFQDNServiceEntry(dk.Name, istio.OperatorComponent)

		virtualService, err := istioClient(t, envConfig.Client().RESTConfig()).NetworkingV1beta1().VirtualServices(dk.Namespace).Get(ctx, serviceName, metav1.GetOptions{})
		require.NoError(t, err, "istio: failed to get '%s' virtual service object", serviceName)

		require.NotEmpty(t, virtualService.OwnerReferences)
		assert.Equal(t, dk.Name, virtualService.OwnerReferences[0].Name)

		require.NotEmpty(t, virtualService.Spec.GetHosts())
		assert.Equal(t, apiHost.Host, virtualService.Spec.GetHosts()[0])

		return ctx
	}
}

func checkServiceEntryForAPIURL(dk dynakube.DynaKube) features.Func { //nolint:dupl
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		apiHost := apiURLCommunicationHost(t, dk)
		serviceName := istio.BuildNameForFQDNServiceEntry(dk.Name, istio.OperatorComponent)

		serviceEntry, err := istioClient(t, envConfig.Client().RESTConfig()).NetworkingV1beta1().ServiceEntries(dk.Namespace).Get(ctx, serviceName, metav1.GetOptions{})
		require.NoError(t, err, "istio: failed to get '%s' service entry object", serviceName)

		require.NotEmpty(t, serviceEntry.OwnerReferences)
		assert.Equal(t, dk.Name, serviceEntry.OwnerReferences[0].Name)

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

func apiURLCommunicationHost(t *testing.T, dk dynakube.DynaKube) istio.CommunicationHost {
	apiHost, err := istio.NewCommunicationHost(dk.APIURL())
	require.NoError(t, err)

	return apiHost
}
