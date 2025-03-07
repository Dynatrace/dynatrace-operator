package statefulset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
)

const (
	// default values
	defaultImageRepo         = "public.ecr.aws/dynatrace/dynatrace-otel-collector"
	defaultImageTag          = "latest"
	containerName            = "collector"
	secretsTokensPath        = "/secrets/tokens"
	otelcSecretTokenFilePath = secretsTokensPath + "/" + consts.OtelcTokenSecretKey
)

func getContainer(dk *dynakube.DynaKube) corev1.Container {
	imageRepo := dk.Spec.Templates.OpenTelemetryCollector.ImageRef.Repository
	imageTag := dk.Spec.Templates.OpenTelemetryCollector.ImageRef.Tag

	if imageRepo == "" {
		imageRepo = defaultImageRepo
	}

	if imageTag == "" {
		imageTag = defaultImageTag
	}

	return corev1.Container{
		Name:            containerName,
		Image:           imageRepo + ":" + imageTag,
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

	if dk.IsExtensionsEnabled() {
		args = append(args, fmt.Sprintf("--config=eec://%s:%d/otcconfig/prometheusMetrics#refresh-interval=5s&auth-file=%s", dk.ExtensionsServiceNameFQDN(), consts.OtelCollectorComPort, otelcSecretTokenFilePath))
	}

	if dk.TelemetryIngest().IsEnabled() {
		args = append(args, "--config=file:///config/telemetry.yaml")
	}

	return args
}
