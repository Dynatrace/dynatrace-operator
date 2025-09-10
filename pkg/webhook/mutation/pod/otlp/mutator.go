package otlp

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/otlp"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	"slices"
)

const (
	envVarOtlpTraceEndpoint    = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	envVarOtlpLogsEndpoint     = "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"
	envVarOtlpMetricsEndpoint  = "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"
	envVarOtlpExporterEndpoint = "OTLP_EXPORTER_OTLP_ENDPOINT"

	envVarOtlpTraceProtocol    = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"
	envVarOtlpLogsProtocol     = "OTEL_EXPORTER_OTLP_LOGS_PROTOCOL"
	envVarOtlpMetricsProtocol  = "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL"
	envVarOtlpExporterProtocol = "OTLP_EXPORTER_OTLP_PROTOCOL"

	envVarOtlpTraceHeaders    = "OTEL_EXPORTER_OTLP_TRACES_HEADERS"
	envVarOtlpLogsHeaders     = "OTEL_EXPORTER_OTLP_LOGS_HEADERS"
	envVarOtlpMetricsHeaders  = "OTEL_EXPORTER_OTLP_METRICS_HEADERS"
	envVarOtlpExporterHeaders = "OTLP_EXPORTER_OTLP_HEADERS"

	envVarOtlpTraceCertificate    = "OTEL_EXPORTER_OTLP_TRACES_CERTIFICATE"
	envVarOtlpLogsCertificate     = "OTEL_EXPORTER_OTLP_LOGS_CERTIFICATE"
	envVarOtlpMetricsCertificate  = "OTEL_EXPORTER_OTLP_METRICS_CERTIFICATE"
	envVarOtlpExporterCertificate = "OTLP_EXPORTER_OTLP_CERTIFICATE"

	// TODO make sure naming does not collide with anything
	activeGateTrustedCertVolumeName    = "activegate-server-certs"
	activeGateTrustedCertSecretKeyPath = "activegate.pem"
	activeGateTrustedCertMountPath     = "activegate-server-certs"
	envActiveGateTrustedCert           = activeGateTrustedCertMountPath + "/" + activeGateTrustedCertSecretKeyPath

	envVarDtApiToken = "DT_API_TOKEN"

	otlpAPIPath = "/v2/otlp"
)

type OTLPMutator struct{}

func NewMutator() dtwebhook.Mutator {
	return &OTLPMutator{}
}

func (OTLPMutator) IsEnabled(request *dtwebhook.BaseRequest) bool {
	//TODO implement when structure for OTLP enrichment config in Dynakube has been defined.
	return true
}

func (OTLPMutator) IsInjected(request *dtwebhook.BaseRequest) bool {
	// TODO the check for already present env vars is also done in the Mutate() implementation, so I'm not sure we need this check here as well.
	// Maybe a check for an annotation indicating that the OTLP vars have already been injected might be better here.
	otlpEndpointEnvVars := []string{envVarOtlpExporterEndpoint, envVarOtlpTraceEndpoint, envVarOtlpMetricsEndpoint, envVarOtlpLogsEndpoint}
	// if any of the OTEL_EXPORTER_* variables are already set in all containers, return false, as we do not want to overwrite them
	// TODO check if also initContainers should be considered
	gotOtlpExporterEnvVars := 0

	for _, c := range request.Pod.Spec.Containers {
		found := false
		for _, e := range c.Env {
			// check if any of the otlp exporter env vars is already present in the container
			if slices.Contains(otlpEndpointEnvVars, e.Name) {
				found = true
				break
			}
		}
		if found {
			gotOtlpExporterEnvVars++
		}
	}
	if gotOtlpExporterEnvVars == len(request.Pod.Spec.Containers) {
		return true
	}
	return false
}

func (OTLPMutator) Mutate(request *dtwebhook.MutationRequest) error {
	injectCert := false
	if request.DynaKube.ActiveGate().HasCaCert() {
		injectCert = true
		// mount the volume with the AG certificate
		defaultMode := int32(420)
		agCertVolume := corev1.Volume{
			Name: activeGateTrustedCertVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &defaultMode,
					SecretName:  consts.OtlpIngestCertsSecretName,
				},
			},
		}
		request.Pod.Spec.Volumes = append(request.Pod.Spec.Volumes, agCertVolume)
	}

	apiURL, err := otlp.GetOtlpIngestEndpoint(&request.DynaKube)
	if err != nil {
		return fmt.Errorf("could not acquire ingest endpoint", err)
	}

	// TODO check if also initContainers should be considered
	for i := range request.Pod.Spec.Containers {
		envVarsToCheck := []string{envVarOtlpExporterEndpoint, envVarOtlpExporterHeaders, envVarOtlpExporterCertificate, envVarOtlpExporterProtocol}
		for _, envVar := range envVarsToCheck {
			// TODO how to handle EnvFrom? OTLP env vars could also be imported via a configMap
			if isEnvVarSet(request.Pod.Spec.Containers[i].Env, envVar) {
				continue
			}
		}
		// TODO use secret specifically for otlp
		// TODO the creation of this secret via the SecretGenerator used by the Dynakube reconciler can take quite some time (~5mins when tested locally), due to the reconciliation interval
		addEnvVarSecretRef(&request.Pod.Spec.Containers[i], envVarDtApiToken, consts.OtlpIngestTokenSecretName, dynatrace.DataIngestToken)

		injectOtlpTraceEnvVars(&request.Pod.Spec.Containers[i], apiURL, injectCert)
		injectOtlpMetricsEnvVars(&request.Pod.Spec.Containers[i], apiURL, injectCert)
		injectOtlpLogsEnvVars(&request.Pod.Spec.Containers[i], apiURL, injectCert)
	}

	return nil
}

func injectOtlpTraceEnvVars(c *corev1.Container, apiURL string, injectCert bool) {
	// check if any environment variable related to the otlp trace exporter is already set.
	// If yes, do not set any related env var to not change any customer specific settings

	envVarsToCheck := []string{envVarOtlpTraceEndpoint, envVarOtlpTraceHeaders, envVarOtlpTraceCertificate, envVarOtlpTraceProtocol}

	for _, envVar := range envVarsToCheck {
		// TODO how to handle EnvFrom? OTLP env vars could also be imported via a configMap
		if isEnvVarSet(c.Env, envVar) {
			return
		}
	}

	addEnvVarLiteralValue(c, envVarOtlpTraceEndpoint, fmt.Sprintf("%s/%s", apiURL, "traces"))
	addEnvVarLiteralValue(c, envVarOtlpTraceProtocol, "http/protobuf")
	addEnvVarLiteralValue(c, envVarOtlpTraceHeaders, fmt.Sprintf("Authorization $(%s)", envVarDtApiToken))

	if injectCert {
		addEnvVarLiteralValue(c, envVarOtlpTraceCertificate, envActiveGateTrustedCert)
	}
}

func injectOtlpMetricsEnvVars(c *corev1.Container, apiURL string, injectCert bool) {
	// check if any environment variable related to the otlp metrics exporter is already set.
	// If yes, do not set any related env var to not change any customer specific settings

	envVarsToCheck := []string{envVarOtlpMetricsEndpoint, envVarOtlpMetricsHeaders, envVarOtlpMetricsCertificate, envVarOtlpMetricsProtocol}

	for _, envVar := range envVarsToCheck {
		// TODO how to handle EnvFrom? OTLP env vars could also be imported via a configMap
		if isEnvVarSet(c.Env, envVar) {
			return
		}
	}

	addEnvVarLiteralValue(c, envVarOtlpMetricsEndpoint, fmt.Sprintf("%s/%s", apiURL, "metrics"))
	addEnvVarLiteralValue(c, envVarOtlpMetricsProtocol, "http/protobuf")
	addEnvVarLiteralValue(c, envVarOtlpMetricsHeaders, fmt.Sprintf("Authorization $(%s)", envVarDtApiToken))

	if injectCert {
		addEnvVarLiteralValue(c, envVarOtlpMetricsCertificate, envActiveGateTrustedCert)
	}
}

func injectOtlpLogsEnvVars(c *corev1.Container, apiURL string, injectCert bool) {
	// check if any environment variable related to the otlp metrics exporter is already set.
	// If yes, do not set any related env var to not change any customer specific settings

	envVarsToCheck := []string{envVarOtlpLogsEndpoint, envVarOtlpLogsHeaders, envVarOtlpLogsCertificate, envVarOtlpLogsProtocol}

	for _, envVar := range envVarsToCheck {
		// TODO how to handle EnvFrom? OTLP env vars could also be imported via a configMap
		if isEnvVarSet(c.Env, envVar) {
			return
		}
	}

	addEnvVarLiteralValue(c, envVarOtlpLogsEndpoint, fmt.Sprintf("%s/%s", apiURL, "logs"))
	addEnvVarLiteralValue(c, envVarOtlpLogsProtocol, "http/protobuf")
	addEnvVarLiteralValue(c, envVarOtlpLogsHeaders, fmt.Sprintf("Authorization $(%s)", envVarDtApiToken))

	if injectCert {
		addEnvVarLiteralValue(c, envVarOtlpLogsCertificate, envActiveGateTrustedCert)
	}
}

func addEnvVarLiteralValue(c *corev1.Container, envVar string, value string) {
	for _, e := range c.Env {
		if e.Name == envVar {
			// do not set the env var if it is already present
			return
		}
	}
	c.Env = append(c.Env, corev1.EnvVar{Name: envVar, Value: value})
}

func addEnvVarSecretRef(c *corev1.Container, envVar, secretName, secretKey string) {
	if isEnvVarSet(c.Env, envVar) {
		return
	}

	c.Env = append(c.Env, corev1.EnvVar{
		Name: envVar,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				Key: secretKey,
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
			},
		}},
	)
}

func isEnvVarSet(env []corev1.EnvVar, envVar string) bool {
	for _, e := range env {
		if e.Name == envVar {
			// do not set the env var if it is already present
			return true
		}
	}
	return false
}

func (OTLPMutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	//TODO implement me
	panic("implement me")
}
