// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package upgrade

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dynakubev1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	e2econst "github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const withCSI = true

func FromAPIToPlatformToken(t *testing.T, releaseTag string) features.Feature {
	builder := features.New("upgrade-from-api-to-platform-token")
	builder.Assess("install operator "+releaseTag, helpers.ToFeatureFunc(operator.Install(releaseTag, withCSI), true))
	secretConfig := tenant.GetSingleTenantSecret(t)

	testDynakube := *componentDynakube.New(
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithCustomPullSecret(e2econst.DevRegistryPullSecretName),
		componentDynakube.WithHostMonitoringSpec(&oneagent.HostInjectSpec{}),
	)

	previousVersionDynakube := &dynakubev1beta5.DynaKube{}
	require.NoError(t, previousVersionDynakube.ConvertFrom(&testDynakube))
	componentDynakube.InstallPreviousVersion(builder, helpers.LevelAssess, &secretConfig, *previousVersionDynakube)

	builder.Assess("update tenant secret to platform token",
		tenant.CreateTenantSecret(secretConfig.PlatformTokens(), testDynakube.Name, testDynakube.Namespace))

	// update to snapshot
	builder.Assess("upgrade operator", helpers.ToFeatureFunc(operator.InstallLocal(withCSI), true))
	componentDynakube.VerifyStartup(builder, features.LevelAssess, testDynakube)
	componentDynakube.VerifyPlatformTokenStatus(builder, testDynakube, true)

	return builder.Feature()
}
