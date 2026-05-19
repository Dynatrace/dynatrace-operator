//go:build e2e

package usepublicregistry

import (
	"context"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	publicRegistryOverrideEnvVar = "E2E_PUBLIC_REGISTRY_OVERRIDE"
)

func publicRegistryOverride(t *testing.T) string {
	t.Helper()
	val := os.Getenv(publicRegistryOverrideEnvVar)
	if val == "" {
		t.Skipf("%s must be set", publicRegistryOverrideEnvVar)
	}

	return val
}

// requireDevRegistrySecret returns a Setup feature.Func that fails the test if
// the devregistry pull secret is not already present in the operator namespace.
// The secret is expected to be provisioned out-of-band on the test cluster (see
// CONTRIBUTING.md); without it, the operator cannot pull images from the
// configured registry, so the test cannot meaningfully run.
func requireDevRegistrySecret() features.Func {
	return k8ssecret.Exists(consts.DevRegistryPullSecretName, operator.DefaultNamespace)
}

func OneAgent(t *testing.T) features.Feature {
	return oneAgentFeature(t, "use-public-registry-oneagent", "use-public-registry-oa", "")
}

func OneAgentWithOverride(t *testing.T) features.Feature {
	return oneAgentFeature(t,
		"use-public-registry-oneagent-with-override",
		"use-public-registry-oa-ovrd",
		publicRegistryOverride(t))
}

func ActiveGate(t *testing.T) features.Feature {
	return activeGateFeature(t, "use-public-registry-activegate", "use-public-registry-ag", "")
}

func ActiveGateWithOverride(t *testing.T) features.Feature {
	return activeGateFeature(t,
		"use-public-registry-activegate-with-override",
		"use-public-registry-ag-ovrd",
		publicRegistryOverride(t))
}

func CodeModules(t *testing.T) features.Feature {
	return codeModulesFeature(t,
		"use-public-registry-codemodules",
		"use-public-registry-cm",
		"use-public-registry-cm-sample",
		"")
}

func CodeModulesWithOverride(t *testing.T) features.Feature {
	return codeModulesFeature(t,
		"use-public-registry-codemodules-with-override",
		"use-public-registry-cm-ovrd",
		"use-public-registry-cm-sample-ovrd",
		publicRegistryOverride(t))
}

func oneAgentFeature(t *testing.T, featureName, dkName, override string) features.Feature {
	builder := features.New(featureName)
	builder.Assess("devregistry pull secret exists", requireDevRegistrySecret())

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakubeComponents.Option{
		dynakubeComponents.WithName(dkName),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
		dynakubeComponents.WithUsePublicRegistryFF(),
		// dev tenants need customPullSecret to be able to pull images from dev ECR registry
		dynakubeComponents.WithCustomPullSecret(consts.DevRegistryPullSecretName),
	}
	if override != "" {
		options = append(options, dynakubeComponents.WithPublicRegistryOverride(override))
	}

	testDynakube := *dynakubeComponents.New(options...)

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)

	builder.Assess("OneAgent DaemonSet ready",
		k8sdaemonset.IsReady(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))
	builder.Assess("OneAgent status reports public-registry source",
		statusSourceIsPublicRegistry(testDynakube, image.OneAgent))

	return builder.Feature()
}

func activeGateFeature(t *testing.T, featureName, dkName, override string) features.Feature {
	builder := features.New(featureName)
	builder.Assess("devregistry pull secret exists", requireDevRegistrySecret())

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakubeComponents.Option{
		dynakubeComponents.WithName(dkName),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithUsePublicRegistryFF(),
		// dev tenants need customPullSecret to be able to pull images from dev ECR registry
		dynakubeComponents.WithCustomPullSecret(consts.DevRegistryPullSecretName),
	}
	if override != "" {
		options = append(options, dynakubeComponents.WithPublicRegistryOverride(override))
	}

	testDynakube := *dynakubeComponents.New(options...)

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)

	const agComponent = "activegate"
	builder.Assess("ActiveGate StatefulSet ready",
		k8sstatefulset.IsReady(activegate.GetActiveGateStateFulSetName(&testDynakube, agComponent), testDynakube.Namespace))
	builder.Assess("ActiveGate status reports public-registry source",
		statusSourceIsPublicRegistry(testDynakube, image.ActiveGate))

	return builder.Feature()
}

func codeModulesFeature(t *testing.T, featureName, dkName, sampleNamespaceName, override string) features.Feature {
	builder := features.New(featureName)
	builder.Assess("devregistry pull secret exists", requireDevRegistrySecret())

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakubeComponents.Option{
		dynakubeComponents.WithName(dkName),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{}),
		dynakubeComponents.WithUsePublicRegistryFF(),
		// dev tenants need customPullSecret to be able to pull images from dev ECR registry
		dynakubeComponents.WithCustomPullSecret(consts.DevRegistryPullSecretName),
		dynakubeComponents.WithAnnotations(map[string]string{
			exp.OANodeImagePullKey: "true",
		}),
	}
	if override != "" {
		options = append(options, dynakubeComponents.WithPublicRegistryOverride(override))
	}

	testDynakube := *dynakubeComponents.New(options...)

	sampleNamespace := *k8snamespace.New(sampleNamespaceName)
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
		// The injected init container image is pulled by kubelet from the user's
		// namespace, so the user's pod must reference the registry pull secret.
		sample.WithImagePullSecret(consts.DevRegistryPullSecretName),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())
	builder.Assess("create registry pull secret in sample namespace",
		copyDevRegistrySecret(sampleNamespace.Name))

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)

	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("CodeModules status reports public-registry source",
		statusSourceIsPublicRegistry(testDynakube, image.CodeModules))

	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}

// copyDevRegistrySecret copies the devregistry pull secret from the operator
// namespace into the sample namespace so the user's pod can authenticate to
// the registry when imagePullSecrets references it.
func copyDevRegistrySecret(targetNamespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		var source corev1.Secret
		require.NoError(t, resources.Get(ctx, consts.DevRegistryPullSecretName, operator.DefaultNamespace, &source))

		target := corev1.Secret{
			Type: source.Type,
			Data: source.Data,
		}
		target.Name = consts.DevRegistryPullSecretName
		target.Namespace = targetNamespace

		err := resources.Create(ctx, &target)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			require.NoError(t, err)
		}

		return ctx
	}
}

// statusSourceIsPublicRegistry refetches the DynaKube and verifies that the
// given component's reported version-source is status.PublicRegistryVersionSource.
// This is the operator's own signal that the use-public-registry FF resolution
// path was taken, so the assertion is identical for the with-override and
// without-override flavors.
func statusSourceIsPublicRegistry(dk dynakube.DynaKube, component image.ComponentType) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		var current dynakube.DynaKube
		require.NoError(t,
			envConfig.Client().Resources().Get(ctx, dk.Name, dk.Namespace, &current))

		var actual status.VersionSource
		switch component {
		case image.OneAgent:
			actual = current.Status.OneAgent.Source
		case image.ActiveGate:
			actual = current.Status.ActiveGate.Source
		case image.CodeModules:
			actual = current.Status.CodeModules.Source
		default:
			require.Failf(t, "unknown component", "unknown component %q", component)

			return ctx
		}

		assert.Equalf(t, status.PublicRegistryVersionSource, actual,
			"expected %s status.source == %q, got %q", component, status.PublicRegistryVersionSource, actual)

		return ctx
	}
}
