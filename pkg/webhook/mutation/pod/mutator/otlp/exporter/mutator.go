package exporter

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	"slices"
)

var (
	log = logd.Get().WithName("otlp-exporter-pod-mutation")
)

type Mutator struct{}

func New() dtwebhook.Mutator {
	return &Mutator{}
}

func (Mutator) IsEnabled(_ *dtwebhook.BaseRequest) bool {
	log.Debug("checking if OTLP env var injection is enabled")

	//TODO implement when structure for OTLP enrichment config in Dynakube has been defined.

	return false
}

func (Mutator) IsInjected(request *dtwebhook.BaseRequest) bool {
	log.Debug("checking if OTLP env vars have already been injected")
	return maputils.GetFieldBool(request.Pod.Annotations, AnnotationInjected, false)
}

func (Mutator) Mutate(request *dtwebhook.MutationRequest) error {
	log.Debug("injecting OTLP env vars")

	apiURL, err := getIngestEndpoint(&request.DynaKube)
	if err != nil {
		return fmt.Errorf("could not acquire ingest endpoint: %w", err)
	}

	for i := range request.Pod.Spec.Containers {
		if dtwebhook.IsContainerExcludedFromInjection(
			request.DynaKube.Annotations,
			request.Pod.Annotations,
			request.Pod.Spec.Containers[i].Name,
		) {
			continue
		}

		envVarsToCheck := []string{OTLPExporterEndpointEnv,
			OTLPExporterHeadersEnv,
			OTLPExporterCertificateEnv,
			OTLPExporterProtocolEnv,
		}

		otlpEnvVarAlreadySet := false
		for _, envVar := range envVarsToCheck {
			if isEnvVarSet(request.Pod.Spec.Containers[i].Env, envVar) {
				otlpEnvVarAlreadySet = true
				break
			}
		}

		if otlpEnvVarAlreadySet {
			continue
		}

		injectTraceEnvVars(&request.Pod.Spec.Containers[i], apiURL)
		injectMetricsEnvVars(&request.Pod.Spec.Containers[i], apiURL)
		injectLogsEnvVars(&request.Pod.Spec.Containers[i], apiURL)
	}
	return nil
}

func injectTraceEnvVars(c *corev1.Container, apiURL string) {
	// check if any environment variable related to the otlp trace exporter is already set.
	// If yes, do not set any related env var to not change any customer specific settings

	envVarsToCheck := []string{
		OTLPTraceEndpointEnv,
		OTLPTraceHeadersEnv,
		OTLPTraceCertificateEnv,
		OTLPTraceProtocolEnv,
	}

	for _, envVar := range envVarsToCheck {
		if isEnvVarSet(c.Env, envVar) {
			return
		}
	}

	addEnvVarLiteralValue(c, OTLPTraceEndpointEnv, fmt.Sprintf("%s/%s", apiURL, "traces"))
	addEnvVarLiteralValue(c, OTLPTraceProtocolEnv, "http/protobuf")
}

func injectMetricsEnvVars(c *corev1.Container, apiURL string) {
	// check if any environment variable related to the otlp trace exporter is already set.
	// If yes, do not set any related env var to not change any customer specific settings

	envVarsToCheck := []string{
		OTLPMetricsEndpointEnv,
		OTLPMetricsHeadersEnv,
		OTLPMetricsCertificateEnv,
		OTLPMetricsProtocolEnv,
	}

	for _, envVar := range envVarsToCheck {
		if isEnvVarSet(c.Env, envVar) {
			return
		}
	}

	addEnvVarLiteralValue(c, OTLPMetricsEndpointEnv, fmt.Sprintf("%s/%s", apiURL, "metrics"))
	addEnvVarLiteralValue(c, OTLPMetricsProtocolEnv, "http/protobuf")
}

func injectLogsEnvVars(c *corev1.Container, apiURL string) {
	// check if any environment variable related to the otlp trace exporter is already set.
	// If yes, do not set any related env var to not change any customer specific settings

	envVarsToCheck := []string{
		OTLPLogsEndpointEnv,
		OTLPLogsHeadersEnv,
		OTLPLogsCertificateEnv,
		OTLPLogsProtocolEnv,
	}

	for _, envVar := range envVarsToCheck {
		if isEnvVarSet(c.Env, envVar) {
			return
		}
	}

	addEnvVarLiteralValue(c, OTLPLogsEndpointEnv, fmt.Sprintf("%s/%s", apiURL, "metrics"))
	addEnvVarLiteralValue(c, OTLPLogsProtocolEnv, "http/protobuf")
}

func addEnvVarLiteralValue(c *corev1.Container, name string, value string) {
	contains := slices.ContainsFunc(c.Env, func(env corev1.EnvVar) bool {
		if env.Name == name {
			return true
		}
		return false
	})
	if !contains {
		c.Env = append(c.Env, corev1.EnvVar{Name: name, Value: value})
	}
}

func getIngestEndpoint(dk *dynakube.DynaKube) (string, error) {
	dtEndpoint := dk.APIURL() + "/v2/otlp"

	if dk.ActiveGate().IsEnabled() {
		tenantUUID, err := dk.TenantUUID()
		if err != nil {
			return "", err
		}

		serviceFQDN := capability.BuildServiceName(dk.Name) + "." + dk.Namespace + ".svc"

		dtEndpoint = fmt.Sprintf("https://%s/e/%s/api/v2/otlp", serviceFQDN, tenantUUID)
	}

	return dtEndpoint, nil
}

func (Mutator) Reinvoke(_ *dtwebhook.ReinvocationRequest) bool {
	log.Debug("reinvocation of OTLP env vars mutator")

	return false
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
