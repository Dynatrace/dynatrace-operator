//go:build e2e

package extensions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("extensions-components-rollout")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithExtensionsPrometheusEnabledSpec(true),
		componentDynakube.WithExtensionsEECImageRefSpec(consts.EecImageRepo, consts.EecImageTag),
		componentDynakube.WithActiveGate(),
		componentDynakube.WithActiveGateTLSSecret(consts.AgSecretName),
		componentDynakube.WithOTelCollectorImageRefSpec(consts.OtelCollectorImageRepo, consts.OtelCollectorImageTag),
	}

	testDynakube := *componentDynakube.New(options...)

	agCrt, err := os.ReadFile(filepath.Join(project.TestDataDir(), consts.AgCertificate))
	require.NoError(t, err)

	agP12, err := os.ReadFile(filepath.Join(project.TestDataDir(), consts.AgCertificateAndPrivateKey))
	require.NoError(t, err)

	agSecret := secret.New(consts.AgSecretName, testDynakube.Namespace,
		map[string][]byte{
			dynakube.ServerCertKey:                 agCrt,
			consts.AgCertificateAndPrivateKeyField: agP12,
		})
	builder.Assess("create AG TLS secret", secret.Create(agSecret))

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("active gate pod is running", activegate.CheckContainer(&testDynakube))

	builder.Assess("extensions execution controller started", statefulset.WaitFor(testDynakube.Extensions().GetExecutionControllerStatefulsetName(), testDynakube.Namespace))

	builder.Assess("extension collector started", statefulset.WaitFor(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	builder.WithTeardown("deleted ag secret", secret.Delete(agSecret))

	return builder.Feature()
}
