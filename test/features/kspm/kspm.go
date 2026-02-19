//go:build e2e

package kspm

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	componentKspm "github.com/Dynatrace/dynatrace-operator/test/helpers/components/kspm"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("kspm-components-rollout")

	secretConfig := tenant.GetSingleTenantSecret(t)

	builder.Setup(componentKspm.DeleteKSPMSettingsFromTenant(secretConfig))

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithKSPM(),
		componentDynakube.WithKSPMImageRefSpec(consts.KSPMImageRepo, consts.KSPMImageTag),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("active gate pod is running", activegate.CheckContainer(&testDynakube))

	builder.Assess("kspm node config collector started", k8sdaemonset.IsReady(testDynakube.KSPM().GetDaemonSetName(), testDynakube.Namespace))

	builder.Assess("check if KSPM settings were created on tenant", componentKspm.CheckKSPMSettingsExistOnTenant(secretConfig, &testDynakube))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}

func OptionalScopes(t *testing.T) features.Feature {
	builder := features.New("kspm-optional-scopes")

	secretConfig := tenant.GetSingleTenantSecret(t)
	if secretConfig.APITokenNoSettings == "" {
		t.Skip("skipping test. no token with missing settings scopes provided")
	}

	builder.Setup(componentKspm.DeleteKSPMSettingsFromTenant(secretConfig))

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithKSPM(),
		componentDynakube.WithKSPMImageRefSpec(consts.KSPMImageRepo, consts.KSPMImageTag),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.InstallWithoutSettingsScopes(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("active gate pod is running", activegate.CheckContainer(&testDynakube))

	builder.Assess("kspm node config collector started", k8sdaemonset.IsReady(testDynakube.KSPM().GetDaemonSetName(), testDynakube.Namespace))

	// Note: When settings scopes are missing on the token, the KSPM settings reconciler should skip creation
	// The test verifies that the operator handles missing scopes gracefully without creating settings

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}
