// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package probes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestReconcileBuildsStatefulSet(t *testing.T) {
	testProbeFunc := func(t *testing.T, probe *corev1.Probe, path string, initialDelaySeconds, periodSeconds, failureThreshold, timeoutSeconds, successThreshold int32) {
		assert.Equal(t, path, probe.HTTPGet.Path)
		assert.Equal(t, int32(9999), probe.HTTPGet.Port.IntVal)
		assert.Equal(t, corev1.URIScheme("HTTPS"), probe.HTTPGet.Scheme)
		assert.Equal(t, initialDelaySeconds, probe.InitialDelaySeconds)
		assert.Equal(t, periodSeconds, probe.PeriodSeconds)
		assert.Equal(t, failureThreshold, probe.FailureThreshold)
		assert.Equal(t, timeoutSeconds, probe.TimeoutSeconds)
		assert.Equal(t, successThreshold, probe.SuccessThreshold)
	}

	t.Run("readiness probe", func(t *testing.T) {
		testProbeFunc(t, BuildReadinessProbe(), "/rest/health", 90, 15, 3, 2, 1)
	})

	t.Run("liveness probe", func(t *testing.T) {
		testProbeFunc(t, BuildLivenessProbe(), "/rest/state", 90, 30, 2, 1, 1)
	})
}
