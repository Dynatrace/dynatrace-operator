// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package permissions

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/deployersamples"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/events"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/environment"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var (
	testEnv env.Environment
	cfg     *envconf.Config
)

func TestMain(m *testing.M) {
	cfg = environment.GetStandardKubeClusterEnvConfig()
	testEnv = env.NewWithConfig(cfg)

	testEnv.AfterEachTest(func(ctx context.Context, c *envconf.Config, t *testing.T) (context.Context, error) {
		if t.Failed() {
			events.LogEvents(ctx, c, t)
		}

		return ctx, nil
	})

	testEnv.Run(m)
}

func TestPermissions_deployer_escalate_no_csi(t *testing.T) {
	testEnv.Test(t, deployersamples.EscalateNoCSI())
}

func TestPermissions_deployer_escalate_with_csi(t *testing.T) {
	testEnv.Test(t, deployersamples.EscalateWithCSI())
}

func TestPermissions_deployer_no_escalate_no_csi(t *testing.T) {
	testEnv.Test(t, deployersamples.NoEscalateNoCSI())
}

func TestPermissions_deployer_no_escalate_with_csi(t *testing.T) {
	testEnv.Test(t, deployersamples.NoEscalateWithCSI())
}
