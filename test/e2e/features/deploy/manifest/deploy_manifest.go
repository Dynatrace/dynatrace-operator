// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package manifest

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/platform"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func NoCSI(t *testing.T) features.Feature {
	return deployFeature(t, false)
}

func CSI(t *testing.T) features.Feature {
	return deployFeature(t, true)
}

func deployFeature(t *testing.T, withCSI bool) features.Feature {
	p, err := platform.NewResolver().GetPlatform()
	require.NoError(t, err)

	name := "deploy-manifest-" + p
	if withCSI {
		name += "-csi"
	}

	builder := features.New(name)

	// InstallLocal also verifies the installation
	builder.Setup(helpers.ToFeatureFunc(operator.InstallLocal(withCSI), true))

	builder.Teardown(helpers.SkipOnFailFast(helpers.ToFeatureFunc(operator.Uninstall(withCSI), true)))

	return builder.Feature()
}
