//go:build e2e

package public_registry

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1"
	dynakubev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	publicRegistrySource = status.VersionSource("public-registry")
	customPullSecretName = "devregistry"
)

var publicRegistryFeatureFlag = map[string]string{dynakubev1beta1.AnnotationFeaturePublicRegistry: "true"}

func Feature(t *testing.T) features.Feature {
	builder := features.New("cloudnative with public registry feature enabled")
	builder.WithLabel("name", "cloudnative-public-registry")
	secretConfig := tenant.GetSingleTenantSecret(t)

	testDynakube := *dynakube.New(
		dynakube.WithCustomPullSecret(customPullSecretName),
		dynakube.WithAnnotations(publicRegistryFeatureFlag),
		dynakube.WithActiveGate(),
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithCloudNativeSpec(&dynakubev1beta1.CloudNativeFullStackSpec{}),
	)

	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	builder.Assess("check dynakube status", checkDynakubeStatus(testDynakube))
	builder.Assess("check whether public registry images are used", checkPublicRegistryUsage(testDynakube))
	builder.Assess("check whether correct image has been downloaded", checkCSIProvisionerEvent(testDynakube))
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	return builder.Feature()
}

func checkDynakubeStatus(dynakube dynakubev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		var dk dynakubev1beta1.DynaKube

		err := dynatracev1beta1.AddToScheme(resources.GetScheme())
		require.NoError(t, err)

		err = resources.Get(ctx, dynakube.Name, dynakube.Namespace, &dk)
		require.NoError(t, err)

		require.NotNil(t, dk.Status.OneAgent)
		assert.NotEmpty(t, dk.Status.OneAgent.VersionStatus.ImageID)
		assert.Equal(t, publicRegistrySource, dk.Status.OneAgent.VersionStatus.Source)
		assert.NotNil(t, dk.Status.OneAgent.VersionStatus.LastProbeTimestamp)

		require.NotNil(t, dk.Status.CodeModules)
		assert.NotEmpty(t, dk.Status.CodeModules.VersionStatus.ImageID)
		assert.Equal(t, publicRegistrySource, dk.Status.CodeModules.VersionStatus.Source)
		assert.NotNil(t, dk.Status.CodeModules.VersionStatus.LastProbeTimestamp)

		require.NotNil(t, dk.Status.ActiveGate)
		assert.NotEmpty(t, dk.Status.ActiveGate.VersionStatus.ImageID)
		assert.Equal(t, publicRegistrySource, dk.Status.ActiveGate.VersionStatus.Source)
		assert.NotNil(t, dk.Status.ActiveGate.VersionStatus.LastProbeTimestamp)

		return ctx
	}
}

func checkPublicRegistryUsage(dynakube dynakubev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		var dk dynakubev1beta1.DynaKube

		err := dynatracev1beta1.AddToScheme(resources.GetScheme())
		require.NoError(t, err)

		err = resources.Get(ctx, dynakube.Name, dynakube.Namespace, &dk)
		require.NoError(t, err)

		oneAgentDaemonSet, err := oneagent.Get(ctx, resources, dynakube)
		require.NoError(t, err)

		require.Equal(t, dk.Status.OneAgent.ImageID, oneAgentDaemonSet.Spec.Template.Spec.Containers[0].Image)

		activeGateStateFulSet, err := activegate.Get(ctx, resources, dynakube)
		require.NoError(t, err)

		require.Equal(t, dk.Status.ActiveGate.ImageID, activeGateStateFulSet.Spec.Template.Spec.Containers[0].Image)

		return ctx
	}
}

func checkCSIProvisionerEvent(dynakube dynakubev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		clientset, err := kubernetes.NewForConfig(resources.GetConfig())
		require.NoError(t, err)

		err = wait.For(func(ctx context.Context) (done bool, err error) {
			events, err := clientset.CoreV1().Events("dynatrace").List(ctx, metav1.ListOptions{
				TypeMeta: metav1.TypeMeta{
					Kind: "Pod",
				},
			})
			require.NoError(t, err)
			for _, event := range events.Items {
				if strings.Contains(event.Message, "Installed agent version: "+dynakube.Status.CodeModules.ImageID) {
					return true, nil
				}
			}
			return false, errors.New("csi-provisioner event not found")
		}, wait.WithTimeout(5*time.Minute))

		require.NoError(t, err)

		return ctx
	}
}
