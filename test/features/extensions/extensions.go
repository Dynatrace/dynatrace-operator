//go:build e2e

package extensions

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("extensions-components-rollout")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithApiUrl(secretConfig.ApiUrl),
		componentDynakube.WithActiveGateTLSSecret(consts.AgSecretName),
		componentDynakube.WithCustomPullSecret(consts.DevRegistryPullSecretName),
		componentDynakube.WithExtensionsEnabledSpec(true),
		componentDynakube.WithExtensionsEECImageRefSpec(consts.EecImageRepo, consts.EecImageTag),
	}

	testDynakube := *componentDynakube.New(options...)

	agCrt, err := os.ReadFile(path.Join(project.TestDataDir(), consts.AgCertificate))
	require.NoError(t, err)

	agP12, err := os.ReadFile(path.Join(project.TestDataDir(), consts.AgCertificateAndPrivateKey))
	require.NoError(t, err)

	agSecret := secret.New(consts.AgSecretName, testDynakube.Namespace,
		map[string][]byte{
			dynakube.TLSCertKey:                    agCrt,
			consts.AgCertificateAndPrivateKeyField: agP12,
		})
	builder.Assess("create AG TLS secret", secret.Create(agSecret))

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("active gate pod is running", checkActiveGateContainer(&testDynakube))

	builder.Assess("extensions execution controller started", statefulset.WaitFor(testDynakube.ExtensionsExecutionControllerStatefulsetName(), testDynakube.Namespace))

	builder.Assess("extension collector started", statefulset.WaitFor(testDynakube.ExtensionsCollectorStatefulsetName(), testDynakube.Namespace))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	builder.WithTeardown("deleted ag secret", secret.Delete(agSecret))

	return builder.Feature()
}

func checkActiveGateContainer(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		var activeGatePod corev1.Pod
		require.NoError(t, resources.WithNamespace(dk.Namespace).Get(ctx, activegate.GetActiveGatePodName(dk, "activegate"), dk.Namespace, &activeGatePod))

		require.NotNil(t, activeGatePod.Spec)
		require.NotEmpty(t, activeGatePod.Spec.Containers)

		return ctx
	}
}
