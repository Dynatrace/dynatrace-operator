// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package objects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func VerifyWorkloadUsesImage(t *testing.T, containers []corev1.Container, expectedImage, workloadName string) {
	for _, c := range containers {
		if c.Image == expectedImage {
			return
		}
	}

	assert.Failf(t, "image not used", "expected image %q not found in workload %q containers", expectedImage, workloadName)
}
