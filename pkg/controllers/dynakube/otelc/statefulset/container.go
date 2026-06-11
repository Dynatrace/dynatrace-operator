package statefulset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	containerName            = "collector"
	secretsTokensPath        = "/secrets/tokens"
	otelcSecretTokenFilePath = secretsTokensPath + "/" + consts.DatasourceTokenSecretKey
)

func getContainer(dk *dynakube.DynaKube, replicas int32) corev1.Container {
	container := corev1.Container{
		Name:            containerName,
		Image:           dk.Spec.Templates.OpenTelemetryCollector.ImageRef.String(),
		ImagePullPolicy: dk.Spec.Templates.OpenTelemetryCollector.ImageRef.GetPullPolicy(),
		SecurityContext: buildSecurityContext(dk),
		Env:             getEnvs(dk, replicas),
		Resources:       dk.Spec.Templates.OpenTelemetryCollector.Resources,
		Args:            buildArgs(dk),
		VolumeMounts:    buildContainerVolumeMounts(dk),
	}

	// Only enable the probes when we control the configuration.
	// When using Prometheus extensions, the EEC sends configuration without health checks.
	// The feature is not GA and may be removed in a future release, so it's an accepted caveat.
	if dk.TelemetryIngest().IsEnabled() {
		container.LivenessProbe = buildLivenessProbe()
		container.ReadinessProbe = buildReadinessProbe()
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

func buildArgs(dk *dynakube.DynaKube) []string {
	args := []string{}

	if ext := dk.Extensions(); ext.IsPrometheusEnabled() {
		args = append(args, fmt.Sprintf("--config=eec://%s:%d/otcconfig/prometheusMetrics#refresh-interval=5s&auth-file=%s", ext.GetServiceNameFQDN(), consts.ExtensionsDatasourceTargetPort, otelcSecretTokenFilePath))
	}

	if dk.TelemetryIngest().IsEnabled() {
		args = append(args, "--config=file:///config/telemetry.yaml")
	}

	return args
}
