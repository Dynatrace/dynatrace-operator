// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"strconv"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsKubemonEnabled(t *testing.T) {
	tests := []struct {
		name          string
		kubemonEnv    string
		kubemonConfig bool
		expected      bool
	}{
		{"disabled when environment variable is unset", "", true, false},
		{"disabled when Kubernetes Monitoring is not configured", "true", false, false},
		{"disabled when environment variable is false", "false", true, false},
		{"enabled when operand and environment variable are enabled", "true", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(k8senv.ExperimentalEnableKubemonOperand, tt.kubemonEnv)
			dk := &DynaKube{}
			if tt.kubemonConfig {
				dk.Spec.KubernetesMonitoring = &kubemon.Spec{}
			}

			assert.Equal(t, tt.expected, dk.IsKubemonEnabled())
		})
	}
}

func TestIsKubernetesMonitoringEnabled(t *testing.T) {
	tests := []struct {
		name          string
		kubemonEnv    string
		kubemonConfig bool
		activeGate    bool
		expected      bool
	}{
		{"disabled when neither operand is enabled", "", false, false, false},
		{"enabled by Kubernetes Monitoring operand", "true", true, false, true},
		{"disabled when Kubernetes Monitoring environment gate is off", "false", true, false, false},
		{"enabled by ActiveGate capability", "", false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(k8senv.ExperimentalEnableKubemonOperand, tt.kubemonEnv)
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
				t.Setenv(k8senv.ExperimentalEnableKubemonOperand, "true")
				dk.Spec.KubernetesMonitoring = &kubemon.Spec{
					Registration: &kubemon.Registration{},
				}
			}

			assert.Equal(t, tt.expect, dk.IsKubernetesMonitoringRegistrationEnabled())
		})
	}
}
