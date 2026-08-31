// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	containerName = "collector"
)

func getContainer(dk *dynakube.DynaKube, replicas int32, imageURI string) corev1.Container {
	container := corev1.Container{
		Name:            containerName,
		Image:           imageURI,
		ImagePullPolicy: dk.Spec.Templates.OpenTelemetryCollector.ImageRef.PullPolicy,
		SecurityContext: buildSecurityContext(dk),
		Env:             getEnvs(dk, replicas),
		Resources:       dk.Spec.Templates.OpenTelemetryCollector.Resources,
		Args:            buildArgs(),
		VolumeMounts:    buildContainerVolumeMounts(dk),
		LivenessProbe:   buildLivenessProbe(),
		ReadinessProbe:  buildReadinessProbe(),
	}

	return container
}

func buildLivenessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt32(otelcgen.ExtensionsHealthCheckPort),
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       30,
		FailureThreshold:    3,
		TimeoutSeconds:      2,
		SuccessThreshold:    1,
	}
}

func buildReadinessProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt32(otelcgen.ExtensionsHealthCheckPort),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       10,
		FailureThreshold:    3,
		TimeoutSeconds:      2,
		SuccessThreshold:    1,
	}
}

func buildArgs() []string {
	return []string{"--config=file:///config/telemetry.yaml"}
}
