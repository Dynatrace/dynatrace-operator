//go:build e2e

package extensions

import (
	"os"
	"path"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	agSecretName                    = "ag-ca"
	agCertificate                   = "custom-cas/agcrt.pem"
	agCertificateAndPrivateKey      = "custom-cas/agcrtkey.p12"
	agCertificateAndPrivateKeyField = "server.p12"
	customPullSecretName            = "azurecr"
	eecImageRepo                    = "extk8sregistry.azurecr.io/eec/dynatrace-eec"
	eecImageTag                     = "1.302.0.20240916-161445"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("extensions-components-rollout")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithApiUrl(secretConfig.ApiUrl),
		componentDynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
		componentDynakube.WithActiveGateTLSSecret(agSecretName),
		componentDynakube.WithCustomPullSecret(customPullSecretName),
		componentDynakube.WithExtensionsEnabledSpec(true),
		componentDynakube.WithExtensionsEECImageRefSpec(eecImageRepo, eecImageTag),
	}

	testDynakube := *componentDynakube.New(options...)

	// Create customPull secret
	customPullSecret := secret.NewDockerConfigJson(customPullSecretName, testDynakube.Namespace, secretConfig.CustomPullSecret)
	builder.Assess("create custom pull secret", secret.Create(customPullSecret))

	agCrt, err := os.ReadFile(path.Join(project.TestDataDir(), agCertificate))
	require.NoError(t, err)

	agP12, err := os.ReadFile(path.Join(project.TestDataDir(), agCertificateAndPrivateKey))
	require.NoError(t, err)

	agSecret := secret.New(agSecretName, testDynakube.Namespace,
		map[string][]byte{
			dynakube.TLSCertKey:             agCrt,
			agCertificateAndPrivateKeyField: agP12,
		})
	builder.Assess("create AG TLS secret", secret.Create(agSecret))

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("extensions execution controller started", statefulset.WaitFor(dynakube.ExtensionsExecutionControllerStatefulsetName, testDynakube.Namespace))

	builder.Assess("extension collector started", statefulset.WaitFor(dynakube.ExtensionsCollectorStatefulsetName, testDynakube.Namespace))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}
