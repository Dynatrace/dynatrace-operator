// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"strconv"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsKubemonEnabled(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		dk := &DynaKube{}
		dk.Spec.KubernetesMonitoring = &kubemon.Spec{}
		assert.True(t, dk.IsKubemonEnabled())
	})

	t.Run("disabled", func(t *testing.T) {
		dk := &DynaKube{}
		assert.False(t, dk.IsKubemonEnabled())
	})
}

func TestIsKubernetesMonitoringEnabled(t *testing.T) {
	tests := []struct {
		name          string
		kubemonConfig bool
		activeGate    bool
		expected      bool
	}{
		{"disabled when neither operand is enabled", false, false, false},
		{"enabled by Kubernetes Monitoring operand", true, false, true},
		{"enabled by ActiveGate capability", false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dk := &DynaKube{}
			if tt.kubemonConfig {
				dk.Spec.KubernetesMonitoring = &kubemon.Spec{}
			}
			if tt.activeGate {
				dk.Spec.ActiveGate = activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				}
			}

			assert.Equal(t, tt.expected, dk.IsKubernetesMonitoringEnabled())
		})
	}
}

func TestIsKubernetesMonitoringRegistrationEnabled(t *testing.T) {
	tests := []struct {
		name         string
		ff           *bool // nil = not set (defaults to true)
		agKubemonCap bool
		kubemonOp    bool
		expect       bool
	}{
		{"false: neither path configured", nil, false, false, false},
		{"true: AG path", nil, true, false, true},
		{"true: AG path with FF on", new(true), true, false, true},
		{"false: AG path with FF off", new(false), true, false, false},
		{"true: kubemon path", nil, false, true, true},
		{"true: kubemon path ignores FF", new(false), false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dk := &DynaKube{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
			}
			if tt.ff != nil {
				dk.Annotations[exp.AGAutomaticK8sAPIMonitoringKey] = strconv.FormatBool(*tt.ff)
			}
			if tt.agKubemonCap {
				dk.Spec.ActiveGate = activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				}
			}
			if tt.kubemonOp {
				dk.Spec.KubernetesMonitoring = &kubemon.Spec{
					Registration: &kubemon.Registration{},
				}
			}

			assert.Equal(t, tt.expect, dk.IsKubernetesMonitoringRegistrationEnabled())
		})
	}
}
