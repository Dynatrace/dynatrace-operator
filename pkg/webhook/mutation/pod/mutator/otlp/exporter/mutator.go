package exporter

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/endpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	activeGateTrustedCertVolumeName = "otlp-dynatrace-certs"
	exporterCertsMountPath          = "/otlp-dynatrace-certs"
)

var (
	log = logd.Get().WithName("otlp-exporter-pod-mutation")
)

type Mutator struct{}

func New() dtwebhook.Mutator {
	return &Mutator{}
}

func (m Mutator) IsEnabled(request *dtwebhook.BaseRequest) bool {
	otlpExporterConfig := request.DynaKube.OTLPExporterConfiguration()

	if !otlpExporterConfig.IsEnabled() {
		log.Debug("OTLP env var injection is disabled", "podName", request.PodName(), "namespace", request.Namespace.Name)

		return false
	}

	log.Debug("OTLP env var injection is enabled", "podName", request.PodName(), "namespace", request.Namespace.Name)

	// first, check if otlp injection is enabled explicitly on pod
	enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationOTLPInjectionEnabled, false)

	if !enabledOnPod {
		// if not enabled explicitly, check general injection setting via 'dynatrace.com/inject' annotation
		enabledOnPod = maputils.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationDynatraceInject, request.DynaKube.FF().IsAutomaticInjection())
	}

	enabledOnNamespace := true

	if otlpExporterConfig.NamespaceSelector.Size() > 0 {
		selector, _ := metav1.LabelSelectorAsSelector(&otlpExporterConfig.NamespaceSelector)

		enabledOnNamespace = selector.Matches(labels.Set(request.Namespace.Labels))
	}

	return enabledOnPod && enabledOnNamespace
}

func (m Mutator) IsInjected(request *dtwebhook.BaseRequest) bool {
	log.Debug("checking if OTLP env vars have already been injected")

	return maputils.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationOTLPInjected, false)
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

	apiURL, err := endpoint.BuildOTLPEndpoint(request.DynaKube)
	if err != nil {
		return false, dtwebhook.MutatorError{
			Err:      fmt.Errorf("could not acquire ingest endpoint: %w", err),
			Annotate: setNotInjectedAnnotationFunc(CouldNotGetIngestEndpointReason),
		}
	}

	// add an environment variable with a secret ref to dynatrace-otlp-exporter-config secret
	dtAPITokenEnvVar := corev1.EnvVar{
		Name: DynatraceAPITokenEnv,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: consts.OTLPExporterSecretName,
				},
				Key: dynatrace.DataIngestToken,
			},
		},
	}

	shouldAddCertificate := request.DynaKube.ActiveGate().IsEnabled() && request.DynaKube.ActiveGate().HasCaCert()

	override := otlpExporterConfig.IsOverrideEnvVarsEnabled()

	// Create per-signal injectors
	injectors := []injector{
		&traceInjector{cfg: otlpExporterConfig},
		&metricsInjector{cfg: otlpExporterConfig},
		&logsInjector{cfg: otlpExporterConfig},
	}

	mutated := false

	for i := range request.Pod.Spec.Containers {
		c := &request.Pod.Spec.Containers[i]

		if shouldSkipContainer(*request, *c, override) {
			continue
		}

		if shouldAddCertificate {
			ensureCertificateVolumeMounted(c)
		}

		// need to add the token env var first so that it can be used in other env vars
		c.Env = env.AddOrUpdate(c.Env, dtAPITokenEnvVar)

		for _, inj := range injectors {
			if inj.Inject(c, apiURL, shouldAddCertificate) {
				mutated = true
			}
		}
	}

	if shouldAddCertificate && mutated {
		addActiveGateCertVolume(request.DynaKube, request.Pod)
	}

	return mutated, nil
}

func ensureCertificateVolumeMounted(c *corev1.Container) {
	alreadyMounted := false

	for _, vm := range c.VolumeMounts {
		if vm.Name == activeGateTrustedCertVolumeName {
			alreadyMounted = true

			break
		}
	}

	if !alreadyMounted {
		c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
			Name:      activeGateTrustedCertVolumeName,
			MountPath: exporterCertsMountPath,
			ReadOnly:  true,
		})
	}
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
		if env.IsIn(c.Env, envVar) {
			return true
		}
	}

	return false
}

func setNotInjectedAnnotationFunc(reason string) func(*corev1.Pod) {
	return func(pod *corev1.Pod) {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		pod.Annotations[dtwebhook.AnnotationOTLPInjected] = "false"
		pod.Annotations[dtwebhook.AnnotationOTLPReason] = reason
	}
}

func addActiveGateCertVolume(dk dynakube.DynaKube, pod *corev1.Pod) {
	if !dk.ActiveGate().IsEnabled() || !dk.ActiveGate().HasCaCert() {
		return
	}

	// avoid duplicate volume additions on reinvocation or multiple container matches
	for _, v := range pod.Spec.Volumes {
		if v.Name == activeGateTrustedCertVolumeName {
			return
		}
	}

	defaultMode := int32(420)
	agCertVolume := corev1.Volume{
		Name: activeGateTrustedCertVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				DefaultMode: &defaultMode,
				SecretName:  consts.OTLPExporterCertsSecretName,
			},
		},
	}

	if pod.Spec.Volumes == nil {
		pod.Spec.Volumes = []corev1.Volume{}
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, agCertVolume)
}
