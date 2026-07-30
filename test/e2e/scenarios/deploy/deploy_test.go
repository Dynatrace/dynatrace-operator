// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package deploy

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/deploy/manifest"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/deploy/permissions"
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

func TestDeploy_permissions_deployer_escalate_no_csi(t *testing.T) {
	testEnv.Test(t, permissions.EscalateNoCSI())
}

func TestDeploy_permissions_deployer_escalate_with_csi(t *testing.T) {
	testEnv.Test(t, permissions.EscalateWithCSI())
}

func TestDeploy_permissions_deployer_no_escalate_no_csi(t *testing.T) {
	testEnv.Test(t, permissions.NoEscalateNoCSI())
}

func TestDeploy_permissions_deployer_no_escalate_with_csi(t *testing.T) {
	testEnv.Test(t, permissions.NoEscalateWithCSI())
}

func TestDeploy_manifest_kubernetes_no_csi(t *testing.T) {
	testEnv.Test(t, manifest.KubernetesNoCSI())
}

func TestDeploy_manifest_kubernetes_csi(t *testing.T) {
	testEnv.Test(t, manifest.KubernetesCSI())
}

func TestDeploy_manifest_openshift_no_csi(t *testing.T) {
	testEnv.Test(t, manifest.OpenshiftNoCSI())
}

func TestDeploy_manifest_openshift_csi(t *testing.T) {
	testEnv.Test(t, manifest.OpenshiftCSI())
}
