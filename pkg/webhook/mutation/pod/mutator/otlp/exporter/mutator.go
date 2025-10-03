package exporter

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	log = logd.Get().WithName("otlp-exporter-pod-mutation")
)

type Mutator struct{}

func New() dtwebhook.Mutator {
	return &Mutator{}
}

func (m Mutator) IsEnabled(request *dtwebhook.BaseRequest) bool {
	log.Debug("checking if OTLP env var injection is enabled", "podName", request.PodName(), "namespace", request.Namespace.Name)

	otlpExporterSpec := request.DynaKube.OTLPExporterConfiguration()

	if !otlpExporterSpec.IsEnabled() {
		log.Debug("OTLP env var injection is disabled", "podName", request.PodName(), "namespace", request.Namespace.Name)

		return false
	}

	// first, check if otlp injection is enabled explicitly on pod
	enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, AnnotationInject, false)

	if !enabledOnPod {
		// if not enabled explicitly, check general injection setting via 'dynatrace.com/inject' annotation
		enabledOnPod = maputils.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationDynatraceInject, request.DynaKube.FF().IsAutomaticInjection())
	}

	namespaceEnabled := true

	if otlpExporterSpec.NamespaceSelector.Size() > 0 {
		selector, _ := metav1.LabelSelectorAsSelector(&otlpExporterSpec.NamespaceSelector)

		namespaceEnabled = selector.Matches(labels.Set(request.Namespace.Labels))
	}

	return enabledOnPod && namespaceEnabled
}

func (m Mutator) IsInjected(request *dtwebhook.BaseRequest) bool {
	log.Debug("checking if OTLP env vars have already been injected")

	return maputils.GetFieldBool(request.Pod.Annotations, AnnotationInjected, false)
}

func (m Mutator) Mutate(request *dtwebhook.MutationRequest) error {
	_, err := m.mutate(request.BaseRequest)

	return err
}

func (m Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	log.Debug("reinvocation of OTLP env vars mutator")

	mutated, err := m.mutate(request.BaseRequest)
	if err != nil {
		log.Error(err, "error during reinvocation of OTLP env vars mutator", "podName", request.PodName(), "namespace", request.Namespace.Name)
	}

	return mutated
}

func (m Mutator) mutate(request *dtwebhook.BaseRequest) (bool, error) {
	otlpExporterConfig := request.DynaKube.OTLPExporterConfiguration()

	if !otlpExporterConfig.IsEnabled() {
		log.Debug(
			"no OTLP exporter configuration set, will not inject OTLP exporter env vars",
			"podName", request.PodName(),
			"namespace", request.Namespace.Name,
		)

		return false, nil
	}

	log.Debug("injecting OTLP env vars", "podName", request.PodName(), "namespace", request.Namespace.Name)

	apiURL, err := getIngestEndpoint(&request.DynaKube)
	if err != nil {
		return false, dtwebhook.MutatorError{
			Err:      fmt.Errorf("could not acquire ingest endpoint: %w", err),
			Annotate: setNotInjectedAnnotationFunc(CouldNotGetIngestEndpointReason),
		}
	}

	override := otlpExporterConfig.IsOverrideEnabled()

	mutated := false

	for i := range request.Pod.Spec.Containers {
		c := &request.Pod.Spec.Containers[i]

		if shouldSkipContainer(*request, *c, override) {
			continue
		}

		if otlpExporterConfig.IsTracesEnabled() && injectTraceEnvVars(c, apiURL, override) {
			mutated = true
		}

		if otlpExporterConfig.IsMetricsEnabled() && injectMetricsEnvVars(c, apiURL, override) {
			mutated = true
		}

		if otlpExporterConfig.IsLogsEnabled() && injectLogsEnvVars(c, apiURL, override) {
			mutated = true
		}
	}

	setInjectedAnnotation(request.Pod)

	return mutated, nil
}

func injectTraceEnvVars(c *corev1.Container, apiURL string, override bool) bool {
	// check if any environment variable related to the otlp trace exporter is already set.
	// If yes, do not set any related env var to not change any customer specific settings
	envVarsToCheck := []string{
		OTLPTraceEndpointEnv,
		OTLPTraceHeadersEnv,
		OTLPTraceCertificateEnv,
		OTLPTraceProtocolEnv,
	}

	if !override {
		for _, envVar := range envVarsToCheck {
			if isEnvVarSet(c.Env, envVar) {
				return false
			}
		}
	}

	addEnvVarLiteralValue(c, OTLPTraceEndpointEnv, fmt.Sprintf("%s/%s", apiURL, "traces"))
	addEnvVarLiteralValue(c, OTLPTraceProtocolEnv, "http/protobuf")

	return true
}

func injectMetricsEnvVars(c *corev1.Container, apiURL string, override bool) bool {
	// check if any environment variable related to the otlp trace exporter is already set.
	// If yes, do not set any related env var to not change any customer specific settings
	envVarsToCheck := []string{
		OTLPMetricsEndpointEnv,
		OTLPMetricsHeadersEnv,
		OTLPMetricsCertificateEnv,
		OTLPMetricsProtocolEnv,
	}

	if !override {
		for _, envVar := range envVarsToCheck {
			if isEnvVarSet(c.Env, envVar) {
				return false
			}
		}
	}

	addEnvVarLiteralValue(c, OTLPMetricsEndpointEnv, fmt.Sprintf("%s/%s", apiURL, "metrics"))
	addEnvVarLiteralValue(c, OTLPMetricsProtocolEnv, "http/protobuf")

	return true
}

func injectLogsEnvVars(c *corev1.Container, apiURL string, override bool) bool {
	// check if any environment variable related to the otlp trace exporter is already set.
	// If yes, do not set any related env var to not change any customer specific settings
	envVarsToCheck := []string{
		OTLPLogsEndpointEnv,
		OTLPLogsHeadersEnv,
		OTLPLogsCertificateEnv,
		OTLPLogsProtocolEnv,
	}

	if !override {
		for _, envVar := range envVarsToCheck {
			if isEnvVarSet(c.Env, envVar) {
				return false
			}
		}
	}

	addEnvVarLiteralValue(c, OTLPLogsEndpointEnv, fmt.Sprintf("%s/%s", apiURL, "logs"))
	addEnvVarLiteralValue(c, OTLPLogsProtocolEnv, "http/protobuf")

	return true
}

func addEnvVarLiteralValue(c *corev1.Container, name string, value string) {
	for i := range c.Env {
		if c.Env[i].Name == name {
			c.Env[i].Value = value

			return
		}
	}

	c.Env = append(c.Env, corev1.EnvVar{Name: name, Value: value})
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

func isEnvVarSet(env []corev1.EnvVar, envVar string) bool {
	for _, e := range env {
		if e.Name == envVar {
			// do not set the env var if it is already present
			return true
		}
	}

	return false
}

func shouldSkipContainer(request dtwebhook.BaseRequest, c corev1.Container, override bool) bool {
	if dtwebhook.IsContainerExcludedFromInjection(
		request.DynaKube.Annotations,
		request.Pod.Annotations,
		c.Name,
	) {
		return true
	}

	if override {
		return false
	}

	envVarsToCheck := []string{
		// general exporter env var
		OTLPExporterEndpointEnv,
		OTLPExporterHeadersEnv,
		OTLPExporterCertificateEnv,
		OTLPExporterProtocolEnv,
		// traces exporter env var
		OTLPTraceEndpointEnv,
		OTLPTraceHeadersEnv,
		OTLPTraceCertificateEnv,
		OTLPTraceProtocolEnv,
		// metrics exporter env var
		OTLPMetricsEndpointEnv,
		OTLPMetricsHeadersEnv,
		OTLPMetricsCertificateEnv,
		OTLPMetricsProtocolEnv,
		// logs exporter env var
		OTLPLogsEndpointEnv,
		OTLPLogsHeadersEnv,
		OTLPLogsCertificateEnv,
		OTLPLogsProtocolEnv,
	}

	for _, envVar := range envVarsToCheck {
		if isEnvVarSet(c.Env, envVar) {
			return true
		}
	}

	return false
}

func setInjectedAnnotation(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[AnnotationInjected] = "true"
	delete(pod.Annotations, AnnotationReason)
}

func setNotInjectedAnnotationFunc(reason string) func(*corev1.Pod) {
	return func(pod *corev1.Pod) {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		pod.Annotations[AnnotationInjected] = "false"
		pod.Annotations[AnnotationReason] = reason
	}
}
