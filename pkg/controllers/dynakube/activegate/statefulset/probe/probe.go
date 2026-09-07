// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Readiness() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/rest/health",
				Port:   intstr.IntOrString{IntVal: agconsts.HTTPSContainerPort},
				Scheme: "HTTPS",
			},
		},
		InitialDelaySeconds: 90,
		PeriodSeconds:       15,
		FailureThreshold:    3,
		TimeoutSeconds:      2,
		SuccessThreshold:    1,
	}
}

func Liveness() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/rest/state",
				Port:   intstr.IntOrString{IntVal: agconsts.HTTPSContainerPort},
				Scheme: "HTTPS",
			},
		},
		InitialDelaySeconds: 90,
		PeriodSeconds:       30,
		FailureThreshold:    2,
		TimeoutSeconds:      1,
		SuccessThreshold:    1,
	}
}
