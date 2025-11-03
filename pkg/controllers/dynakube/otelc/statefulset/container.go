package statefulset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
)

const (
	containerName            = "collector"
	secretsTokensPath        = "/secrets/tokens"
	otelcSecretTokenFilePath = secretsTokensPath + "/" + consts.DatasourceTokenSecretKey
)

func getContainer(dk *dynakube.DynaKube) corev1.Container {
	return corev1.Container{
		Name:            containerName,
		Image:            dk.Spec.Templates.OpenTelemetryCollector.ImageRef.String(),
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: buildSecurityContext(),
		Env:             getEnvs(dk),
		Resources:       dk.Spec.Templates.OpenTelemetryCollector.Resources,
		Args:            buildArgs(dk),
		VolumeMounts:    buildContainerVolumeMounts(dk),
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
