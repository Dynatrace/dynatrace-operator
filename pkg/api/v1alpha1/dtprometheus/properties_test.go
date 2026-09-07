// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDTPrometheus_Conditions(t *testing.T) {
	dtp := &DTPrometheus{}

	conditions := dtp.Conditions()
	*conditions = append(*conditions, metav1.Condition{Type: "Ready"})

	assert.Same(t, &dtp.Status.Conditions, dtp.Conditions())
	assert.Len(t, dtp.Status.Conditions, 1)
	assert.Equal(t, "Ready", dtp.Status.Conditions[0].Type)
}
