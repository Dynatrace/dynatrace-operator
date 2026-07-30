// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package manifest

import (
	"context"
	"strconv"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/platform"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func KubernetesNoCSI() features.Feature {
	return feature("kubernetes", false)
}

func KubernetesCSI() features.Feature {
	return feature("kubernetes", true)
}

func OpenshiftNoCSI() features.Feature {
	return feature("openshift", false)
}

func OpenshiftCSI() features.Feature {
	return feature("openshift", true)
}

func feature(platform string, withCSI bool) features.Feature {
	builder := features.New("deploy-manifest-" + platform + "-csi-" + strconv.FormatBool(withCSI))

	builder.Setup(checkPlatform(platform))
	builder.Setup(installManifests(platform, withCSI))
	builder.Assess("operator installed", verifyInstall(withCSI))
	builder.Teardown(uninstallManifests(platform, withCSI))

	return builder.Feature()
}

func checkPlatform(wantPlatform string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		isOpenshift, err := platform.NewResolver().IsOpenshift()
		require.NoError(t, err, "failed to detect cluster platform")

		if wantPlatform == "openshift" && !isOpenshift {
			t.Skip("skipping openshift manifests, cluster is not openshift")
		}

		if wantPlatform == "kubernetes" && isOpenshift {
			t.Skip("skipping kubernetes manifests, cluster is openshift")
		}

		return ctx
	}
}

func installManifests(platform string, withCSI bool) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		err := operator.InstallViaManifests(platform, withCSI)
		require.NoError(t, err, "failed to apply %s manifests (csi=%t)", platform, withCSI)

		return ctx
	}
}

func verifyInstall(withCSI bool) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		ctx, err := operator.VerifyInstall(ctx, c, withCSI)
		require.NoError(t, err, "operator installation verification failed")

		return ctx
	}
}

func uninstallManifests(platform string, withCSI bool) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		if c.FailFast() {
			return ctx
		}

		err := operator.UninstallViaManifests(platform, withCSI)
		require.NoError(t, err, "manifest cleanup failed")

		return ctx
	}
}
