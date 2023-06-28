//go:build e2e

package public_registry

import (
	"context"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	publicRegistrySource = dynatracev1beta1.VersionSource("public-registry")
	customPullSecretName = "devregistry"
)

var publicRegistryFeatureFlag = map[string]string{dynatracev1beta1.AnnotationFeaturePublicRegistry: "true"}

func publicRegistry(t *testing.T) features.Feature {
	builder := features.New("cloudnative with public registry feature enabled")
	secretConfig := tenant.GetSingleTenantSecret(t)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		WithCustomPullSecret(customPullSecretName).
		WithDynakubeNamespaceSelector().
		WithAnnotations(publicRegistryFeatureFlag).
		WithActiveGate().
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(&dynatracev1beta1.CloudNativeFullStackSpec{})
	testDynakube := dynakubeBuilder.Build()

	// Register operator + dynakube install
	assess.InstallDynatrace(builder, &secretConfig, testDynakube)

	builder.Assess("check dynakube status", checkDynakubeStatus(testDynakube))
	builder.Assess("check whether public registry images are used", checkPublicRegistryUsage(testDynakube))
	builder.Assess("check whether correct image has been downloaded", checkCSIProvisionerEvent(testDynakube))

	// Register dynakube and operator uninstall
	teardown.UninstallDynatrace(builder, testDynakube)

	return builder.Feature()
}

func checkDynakubeStatus(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		var dk dynatracev1beta1.DynaKube

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

func checkPublicRegistryUsage(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		var dk dynatracev1beta1.DynaKube

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

func checkCSIProvisionerEvent(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		clientset, err := kubernetes.NewForConfig(resources.GetConfig())
		require.NoError(t, err)

		err = wait.For(func() (done bool, err error) {
			events, err := clientset.CoreV1().Events("dynatrace").List(ctx, v1.ListOptions{
				TypeMeta: v1.TypeMeta{
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
		})

		require.NoError(t, err)

		return ctx
	}
}
