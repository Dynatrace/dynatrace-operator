// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPrometheusMonitoring_Conditions(t *testing.T) {
	pm := &PrometheusMonitoring{}

	conditions := pm.Conditions()
	*conditions = append(*conditions, metav1.Condition{Type: "Ready"})

	assert.Same(t, &pm.Status.Conditions, pm.Conditions())
	assert.Len(t, pm.Status.Conditions, 1)
	assert.Equal(t, "Ready", pm.Status.Conditions[0].Type)
}
