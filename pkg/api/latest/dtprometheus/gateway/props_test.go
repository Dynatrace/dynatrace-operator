// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

func TestNew(t *testing.T) {
	spec := &Spec{}

	gateway := New(spec, "dtprom")

	assert.Same(t, spec, gateway.Spec)
	assert.Equal(t, "dtprom"+NameSuffix, gateway.GetName())
}

func TestGateway_GetName(t *testing.T) {
	gateway := New(&Spec{}, "dtprom")

	assert.Equal(t, "dtprom-gateway", gateway.GetName())
}

func TestGateway_SetName(t *testing.T) {
	gateway := New(&Spec{}, "old")

	gateway.SetName("new")

	assert.Equal(t, "new-gateway", gateway.GetName())
}

func TestGateway_GetUpdateStrategy(t *testing.T) {
	strategy := appsv1.StatefulSetUpdateStrategy{
		Type: appsv1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
			Partition: new(int32(0)),
		},
	}

	gateway := New(&Spec{UpdateStrategy: strategy}, "dtprom")

	assert.Equal(t, strategy, gateway.GetUpdateStrategy())
}

// TestGateway_PromotesCommonGetters verifies that the shared common.Spec getters
// are promoted through the wrapper's embedded *Spec.
func TestGateway_PromotesCommonGetters(t *testing.T) {
	gateway := New(&Spec{}, "dtprom")
	assert.Equal(t, int32(2), gateway.GetReplicas())

	gateway.Replicas = new(int32(3))
	assert.Equal(t, int32(3), gateway.GetReplicas())
}
