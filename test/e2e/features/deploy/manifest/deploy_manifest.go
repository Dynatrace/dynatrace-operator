// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package manifest

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/platform"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	withCSI    = true
	withoutCSI = false
	k8s        = "kubernetes"
	ocp        = "openshift"
)

func KubernetesNoCSI(t *testing.T) features.Feature { //nolint:dupl
	builder := features.New("deploy-manifest-k8s")

	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err, "failed to detect cluster platform")

	if isOpenshift {
		t.Skip("skipping kubernetes manifests, cluster is openshift")
	}

	builder.Setup(helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		return ctx, operator.InstallViaManifests(k8s, withoutCSI)
	}, true))

	builder.Assess("operator installed", helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		return operator.VerifyInstall(ctx, c, withoutCSI)
	}, true))

	builder.Teardown(helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		if c.FailFast() {
			return ctx, nil
		}

		return ctx, operator.UninstallViaManifests(k8s, withoutCSI)
	}, true))

	return builder.Feature()
}

func KubernetesCSI(t *testing.T) features.Feature { //nolint:dupl
	builder := features.New("deploy-manifest-k8s-csi")

	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err, "failed to detect cluster platform")

	if isOpenshift {
		t.Skip("skipping kubernetes manifests, cluster is openshift")
	}

	builder.Setup(helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		return ctx, operator.InstallViaManifests(k8s, withCSI)
	}, true))

	builder.Assess("operator installed", helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		return operator.VerifyInstall(ctx, c, withCSI)
	}, true))

	builder.Teardown(helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		if c.FailFast() {
			return ctx, nil
		}

		return ctx, operator.UninstallViaManifests(k8s, withCSI)
	}, true))

	return builder.Feature()
}

func OpenshiftNoCSI(t *testing.T) features.Feature { //nolint:dupl
	builder := features.New("deploy-manifest-ocp")

	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err, "failed to detect cluster platform")

	if !isOpenshift {
		t.Skip("skipping openshift manifests, cluster is kubernetes")
	}

	builder.Setup(helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		return ctx, operator.InstallViaManifests(ocp, withoutCSI)
	}, true))

	builder.Assess("operator installed", helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		return operator.VerifyInstall(ctx, c, withoutCSI)
	}, true))

	builder.Teardown(helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		if c.FailFast() {
			return ctx, nil
		}

		return ctx, operator.UninstallViaManifests(ocp, withoutCSI)
	}, true))

	return builder.Feature()
}

func OpenshiftCSI(t *testing.T) features.Feature { //nolint:dupl
	builder := features.New("deploy-manifest-ocp-csi")

	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err, "failed to detect cluster platform")

	if !isOpenshift {
		t.Skip("skipping openshift manifests, cluster is kubernetes")
	}

	builder.Setup(helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		return ctx, operator.InstallViaManifests(ocp, withCSI)
	}, true))

	builder.Assess("operator installed", helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		return operator.VerifyInstall(ctx, c, withCSI)
	}, true))

	builder.Teardown(helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
		if c.FailFast() {
			return ctx, nil
		}

		return ctx, operator.UninstallViaManifests(ocp, withCSI)
	}, true))

	return builder.Feature()
}
