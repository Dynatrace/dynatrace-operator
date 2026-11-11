// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelfMonitoringDisabled(t *testing.T) {
	// The shared test helpers opt every DynaKube into self-monitoring, so the validator is called directly here.
	validator := &Validator{apiReader: fake.NewClient(), modules: installconfig.GetModules()}

	t.Run("warns when self-monitoring is not configured", func(t *testing.T) {
		dk := &dynakube.DynaKube{ObjectMeta: defaultDynakubeObjectMeta, Spec: dynakube.DynaKubeSpec{APIURL: testAPIURL}}

		warnings, err := validator.ValidateCreate(t.Context(), dk)
		require.NoError(t, err)
		assert.Equal(t, []string{warningSelfMonitoringDisabled}, []string(warnings))
	})

	t.Run("warns when self-monitoring is removed", func(t *testing.T) {
		oldDK := &dynakube.DynaKube{ObjectMeta: defaultDynakubeObjectMeta, Spec: dynakube.DynaKubeSpec{APIURL: testAPIURL, SelfMonitoring: &dynakube.SelfMonitoringSpec{}}}
		newDK := &dynakube.DynaKube{ObjectMeta: defaultDynakubeObjectMeta, Spec: dynakube.DynaKubeSpec{APIURL: testAPIURL}}

		warnings, err := validator.ValidateUpdate(t.Context(), oldDK, newDK)
		require.NoError(t, err)
		assert.Contains(t, warnings, warningSelfMonitoringDisabled)
	})

	t.Run("no warning when self-monitoring is configured", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL, SelfMonitoring: &dynakube.SelfMonitoringSpec{}},
		})
	})
}
