// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package kspm

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	componentKspm "github.com/Dynatrace/dynatrace-operator/test/helpers/components/kspm"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("kspm-components-rollout")

	secretConfig := tenant.GetSingleTenantSecret(t)

	builder.Setup(componentKspm.DeleteKSPMSettingsFromTenant(secretConfig))

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithKSPM(),
		componentDynakube.WithActiveGate(),
	}

	if !tenant.UsePhase3Tenant() {
		// Gen2 tenants don't serve the NCC image from the fleet management endpoint, so keep using the pinned image.
		options = append(options, componentDynakube.WithKSPMImageRef(t, componentDynakube.GetLatestKSPMImageTagURI(t)))
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("active gate pod is running", activegate.CheckContainer(&testDynakube))

	builder.Assess("kspm node config collector started", k8sdaemonset.IsReady(testDynakube.KSPM().GetDaemonSetName(), testDynakube.Namespace))

	builder.Assess("kspm node config collector uses resolved image", kspmUsesResolvedImage(&testDynakube))

	builder.Assess("check if KSPM settings were created on tenant", componentKspm.CheckKSPMSettingsExistOnTenant(secretConfig, &testDynakube))

	return builder.Feature()
}

func FeatureWithKubemon(t *testing.T) features.Feature {
	builder := features.New("kspm-with-kubernetes-monitoring")

	secretConfig := tenant.GetSingleTenantSecret(t)

	builder.Setup(componentKspm.DeleteKSPMSettingsFromTenant(secretConfig))

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithKSPM(),
		componentDynakube.WithKSPMImageRef(t, componentDynakube.GetLatestKSPMImageTagURI(t)),
		componentDynakube.WithKubernetesMonitoringRegistration(),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("kubemon statefulset is ready", k8sstatefulset.WaitFor(testDynakube.KubernetesMonitoring().GetStatefulSetName(), testDynakube.Namespace))

	builder.Assess("kspm node config collector started", k8sdaemonset.IsReady(testDynakube.KSPM().GetDaemonSetName(), testDynakube.Namespace))

	builder.Assess("check if KSPM settings were created on tenant", componentKspm.CheckKSPMSettingsExistOnTenant(secretConfig, &testDynakube))

	return builder.Feature()
}

// OptionalScopes verifies that the operator handles missing settings scopes gracefully without creating KSPM settings on the tenant.
//
// Note: When settings scopes are missing on the token, the KSPM settings reconciler should skip the creation.
func OptionalScopes(t *testing.T) features.Feature {
	builder := features.New("kspm-with-optional-scopes")

	secretConfig := tenant.GetSingleTenantSecret(t)
	if secretConfig.APITokenNoSettings == "" && secretConfig.PlatformTokenNoSettings == "" {
		t.Skip("skipping test. no token with missing settings scopes provided")
	}

	builder.Setup(componentKspm.DeleteKSPMSettingsFromTenant(secretConfig))

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithKSPM(),
		componentDynakube.WithKSPMImageRef(t, componentDynakube.GetLatestKSPMImageTagURI(t)),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.InstallWithoutSettingsScopes(builder, &secretConfig, testDynakube)

	builder.Assess("active gate pod is running", activegate.CheckContainer(&testDynakube))

	builder.Assess("kspm node config collector started", k8sdaemonset.IsReady(testDynakube.KSPM().GetDaemonSetName(), testDynakube.Namespace))

	return builder.Feature()
}

func kspmUsesResolvedImage(dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		var current dynakube.DynaKube
		require.NoError(t, envConfig.Client().Resources().Get(ctx, dk.Name, dk.Namespace, &current))

		require.NotEmpty(t, current.Status.KSPM.ResolvedImage)

		return k8sdaemonset.VerifyUsesImage(current.KSPM().GetDaemonSetName(), current.Namespace, current.Status.KSPM.ResolvedImage)(ctx, t, envConfig)
	}
}
