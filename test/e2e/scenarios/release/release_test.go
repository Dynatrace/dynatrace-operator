// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package release

import (
	"context"
	"testing"

	cloudnativeupgrade "github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative/upgrade"
	extensionsupgrade "github.com/Dynatrace/dynatrace-operator/test/e2e/features/extensions/upgrade"
	tokenupgrade "github.com/Dynatrace/dynatrace-operator/test/e2e/features/token/upgrade"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
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
	releaseTag16 = "1.6.3"
	releaseTag17 = "1.7.3"
	releaseTag18 = "1.8.1"
	releaseTag19 = "1.9.0"
)

func TestMain(m *testing.M) {
	cfg = environment.GetStandardKubeClusterEnvConfig()
	testEnv = env.NewWithConfig(cfg)
	testEnv.Setup(helpers.SetScheme)

	testEnv.BeforeEachTest(func(ctx context.Context, envConfig *envconf.Config, t *testing.T) (context.Context, error) {
		// TODO Remove this after 1.10 release
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

		// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
		if !envConfig.FailFast() {
			return operator.Uninstall(true)(ctx, envConfig)
		}

		return ctx, nil
	})

	testEnv.Run(m)
}

func TestRelease_cloudnative_upgrade_19(t *testing.T) {
	testEnv.Test(t, cloudnativeupgrade.Feature(t, releaseTag19))
}

func TestRelease_cloudnative_upgrade_18(t *testing.T) {
	testEnv.Test(t, cloudnativeupgrade.Feature(t, releaseTag18))
}

func TestRelease_cloudnative_upgrade_17(t *testing.T) {
	testEnv.Test(t, cloudnativeupgrade.Feature(t, releaseTag17))
}

func TestRelease_cloudnative_upgrade_16(t *testing.T) {
	testEnv.Test(t, cloudnativeupgrade.Feature(t, releaseTag16))
}

func TestRelease_extensions_upgrade(t *testing.T) {
	testEnv.Test(t, extensionsupgrade.Feature(t, releaseTag19))
}

func TestRelease_platform_token_upgrade(t *testing.T) {
	testEnv.Test(t, tokenupgrade.FromAPIToPlatformToken(t, releaseTag19))
}
