// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package release

import (
	"context"
	"testing"

	tokenupgrade "github.com/Dynatrace/dynatrace-operator/test/e2e/features/token/upgrade"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/upgrade"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/events"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/environment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/logs"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var (
	testEnv env.Environment
	cfg     *envconf.Config
)

const (
	releaseTag18  = "1.8.1"
	releaseTag19  = "1.9.0"
	releaseTag110 = "1.10.2"
)

func TestMain(m *testing.M) {
	cfg = environment.GetStandardKubeClusterEnvConfig()
	testEnv = env.NewWithConfig(cfg)
	testEnv.Setup(helpers.SetScheme)

	testEnv.BeforeEachTest(func(ctx context.Context, envConfig *envconf.Config, t *testing.T) (context.Context, error) {
		if tenant.UsePlatformToken() {
			t.Skip("skip test from platform token")
		}

		return ctx, nil
	})

	testEnv.AfterEachTest(func(ctx context.Context, envConfig *envconf.Config, t *testing.T) (context.Context, error) {
		if t.Failed() {
			events.LogEvents(ctx, envConfig, t)
			logs.WriteOperatorLogToFile(ctx, envConfig, t)
		}

		return ctx, nil
	})

	testEnv.Run(m)
}

func TestRelease_helm_upgrade_110(t *testing.T) {
	testEnv.Test(t, upgrade.Feature(t, releaseTag110))
}

func TestRelease_helm_upgrade_19(t *testing.T) {
	testEnv.Test(t, upgrade.Feature(t, releaseTag19))
}

func TestRelease_helm_upgrade_18(t *testing.T) {
	testEnv.Test(t, upgrade.Feature(t, releaseTag18))
}

func TestRelease_platform_token_upgrade(t *testing.T) {
	testEnv.Test(t, tokenupgrade.FromAPIToPlatformToken(t, releaseTag19))
}

func TestRelease_manifest_upgrade_110(t *testing.T) {
	testEnv.Test(t, upgrade.ManifestFeature(t, releaseTag110))
}

func TestRelease_manifest_upgrade_19(t *testing.T) {
	testEnv.Test(t, upgrade.ManifestFeature(t, releaseTag19))
}

func TestRelease_manifest_upgrade_18(t *testing.T) {
	testEnv.Test(t, upgrade.ManifestFeature(t, releaseTag18))
}
