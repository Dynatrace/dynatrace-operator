// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"text/template"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	otlpendpoint "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/endpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// gatewayConfigKey is the ConfigMap data key holding the rendered OTel Collector config,
	// mounted as relay.yaml (see --config=/conf/relay.yaml on the container).
	gatewayConfigKey = "relay"

	// userManagedAnnotation on the ConfigMap opts it out of operator-managed updates.
	userManagedAnnotation = "internal.operator.dynatrace.com/user-managed"

	trustedCAVolumeMountPath      = "/tls/custom/cacerts"
	trustedCAFile                 = "rootca.pem"
	activeGateCertVolumeMountPath = "/tls/custom/activegate"
	activeGateCertFile            = "cert.pem"

	otlpPortName     = "otlp"
	otlpPort         = 4317
	otlpHTTPPortName = "otlp-http"
	otlpHTTPPort     = 4318

	healthCheckPort = 13133

	serviceAccountName = "dynatrace-prometheus-gateway"

	configVolumeName  = "opentelemetry-collector-configmap"
	configMountDir    = "/conf"
	relayConfigFile   = "relay.yaml"
	agCertVolumeName  = "activegate-cert"
	cacertsVolumeName = "cacerts"

	// configHashAnnotation on the pod template drives rolling restarts when the rendered
	// config content changes.
	configHashAnnotation = "internal.operator.dynatrace.com/gateway-config-hash"
)

type Reconciler struct {
	client.Client
}

type reconcileScope struct {
	// Required for reconcile
	Owner       *dtprometheus.DTPrometheus
	DynaKube    *dynakube.DynaKube
	Spec        *dtprometheus.Gateway
	AppLabels   *k8slabel.Labels
	ImageClient image.Client
	// Computed during reconcile
	resolvedImage string
	ConfigMapHash string
	StatefulSet   *appsv1.StatefulSet
}

func (r *Reconciler) Reconcile(ctx context.Context, dtp *dtprometheus.DTPrometheus, dk *dynakube.DynaKube, imageClient image.Client) error {
	ctx, _ = logd.NewFromContext(ctx, "gateway")

	scope := &reconcileScope{
		Owner:       dtp,
		DynaKube:    dk,
		Spec:        dtp.Gateway(),
		AppLabels:   k8slabel.New("opentelemetry-gateway", "otel-gateway", ""),
		ImageClient: imageClient,
	}

	var err error

	for _, f := range []func(context.Context, *reconcileScope) error{
		r.reconcileConfigMap,
		r.reconcileStatefulset,
		r.reconcileService,
	} {
		if err = f(ctx, scope); err != nil {
			break
		}
	}

	r.reconcileCondition(scope, err)

	return err
}

// resolveImage requires an explicit image for now: the fleet image API is not ready yet for
// the gateway component (same caveat as the target allocator reconciler).
func (r *Reconciler) resolveImage(s *reconcileScope) error {
	if s.Spec.Image == "" {
		// TODO: fix this once the gateway image is available from the fleet management API
		// imageURI, err := registry.ResolveImage(ctx, s.ImageClient, s.Owner.Spec.PublicRegistryOverride, image.Gateway)
		return errors.New("missing image")
	}

	s.resolvedImage = s.Spec.Image

	return nil
}

func (r *Reconciler) reconcileCondition(s *reconcileScope, err error) {
	condition := metav1.Condition{Type: dtprometheus.GatewayAvailable}

	switch {
	case err != nil:
		condition.Status = metav1.ConditionFalse
		condition.Reason = status.ReasonError
		condition.Message = safeUnwrap(err).Error()
	case k8sstatefulset.IsRolloutComplete(s.StatefulSet):
		condition.Status = metav1.ConditionTrue
		condition.Reason = status.ReasonAvailable
		condition.Message = "gateway is ready"
	default:
		condition.Status = metav1.ConditionFalse
		condition.Reason = status.ReasonReconciling
		condition.Message = "gateway is pending"
	}

	_ = meta.SetStatusCondition(&s.Owner.Status.Conditions, condition)
}

// safeUnwrap returns the innermost wrapped error, so the condition message shows
// the root cause instead of the "reconcile x: ..." wrapping added by the callers.
func safeUnwrap(err error) error {
	if u := errors.Unwrap(err); u != nil {
		return u
	}

	return err
}

// mergeAppLabels merges the standard app labels onto obj's existing labels, keeping any
// custom labels already present.
func mergeAppLabels(obj client.Object, appLabels *k8slabel.Labels) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	maps.Copy(labels, appLabels.AsMap())
	obj.SetLabels(labels)
}

// createOrUpdate wraps RetryCreateOrUpdate + SetControllerReference + created/updated logging,
// shared by the configmap/statefulset/service reconcile steps.
func (r *Reconciler) createOrUpdate(ctx context.Context, owner metav1.Object, obj client.Object, kind string, mutate func() error) error {
	log := logd.FromContext(ctx)

	result, err := k8sobject.RetryCreateOrUpdate(ctx, r, obj, func() error {
		if err := mutate(); err != nil {
			return err
		}

		return controllerutil.SetControllerReference(owner, obj, r.Scheme())
	})
	if err != nil {
		return fmt.Errorf("reconcile %s: %w", kind, err)
	}

	switch result {
	case controllerutil.OperationResultCreated:
		log.Info("created " + kind)
	case controllerutil.OperationResultUpdated:
		log.Info("updated " + kind)
	}

	return nil
}

func (r *Reconciler) reconcileConfigMap(ctx context.Context, s *reconcileScope) error {
	log := logd.FromContext(ctx)

	name := s.Spec.GetStatefulSetName()

	existing := &corev1.ConfigMap{}
	if err := r.Get(ctx, client.ObjectKey{Name: name, Namespace: s.Owner.Namespace}, existing); err == nil {
		if existing.Annotations[userManagedAnnotation] == "true" {
			log.Info("skipping user-managed configmap")

			checksum := sha256.Sum256([]byte(existing.Data[gatewayConfigKey]))
			s.ConfigMapHash = hex.EncodeToString(checksum[:])

			return nil
		}
	} else if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("get configmap: %w", err)
	}

	data, err := buildGatewayConfigData(s.DynaKube)
	if err != nil {
		return fmt.Errorf("build gateway config data: %w", err)
	}

	rendered, err := renderGatewayConfig(data)
	if err != nil {
		return fmt.Errorf("render gateway config: %w", err)
	}

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.Owner.Namespace}}

	err = r.createOrUpdate(ctx, s.Owner, cm, "configmap", func() error {
		mergeAppLabels(cm, s.AppLabels)
		cm.Data = map[string]string{gatewayConfigKey: rendered}

		return nil
	})
	if err != nil {
		return err
	}

	checksum := sha256.Sum256([]byte(rendered))
	s.ConfigMapHash = hex.EncodeToString(checksum[:])

	return nil
}

// gatewayConfigData feeds the OTel Collector relay.yaml template.
type gatewayConfigData struct {
	Endpoint              string
	HasCustomCA           bool
	CustomCAPath          string
	ResourceAttributes    map[string]string
	HasResourceAttributes bool
}

// buildGatewayConfigData resolves the DynaKube-derived inputs to the relay.yaml template.
// The endpoint and CA-trust decisions (AG-routed vs. direct, which CA to mount) are reused
// from the DynaKube's own otelc component (otelc/endpoint.BuildOTLPEndpoint,
// dk.IsAGCertificateNeeded/IsCACertificateNeeded) so both components stay in lockstep on
// that decision; the config rendering itself is a plain template, not otelcgen.
func buildGatewayConfigData(dk *dynakube.DynaKube) (gatewayConfigData, error) {
	dtEndpoint, err := otlpendpoint.BuildOTLPEndpoint(*dk)
	if err != nil {
		return gatewayConfigData{}, err
	}

	data := gatewayConfigData{Endpoint: dtEndpoint}

	switch {
	case dk.IsAGCertificateNeeded():
		data.HasCustomCA = true
		data.CustomCAPath = activeGateCertVolumeMountPath + "/" + activeGateCertFile
	case dk.IsCACertificateNeeded():
		data.HasCustomCA = true
		data.CustomCAPath = trustedCAVolumeMountPath + "/" + trustedCAFile
	}

	if attrs := dk.GetResourceAttributes(); len(attrs) > 0 {
		data.ResourceAttributes = attrs
		data.HasResourceAttributes = true
	}

	return data, nil
}

// gatewayConfigTemplate is a best-effort relay.yaml skeleton: receiver/exporter/extension
// wiring and the conditional branches (custom CA, resource attributes) are solid, but the
// exact k8s_attributes/transform sub-config is a placeholder pending the real POC config.
const gatewayConfigTemplate = `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: ${env:MY_POD_IP}:4317
      http:
        endpoint: ${env:MY_POD_IP}:4318

processors:
  memory_limiter:
    check_interval: 1s
    limit_percentage: 80
    spike_limit_percentage: 20
  metric_start_time: {}
  cumulativetodelta: {}
  k8s_attributes: {}
  transform: {}
{{- if .HasResourceAttributes }}
  resource/dynakube:
    attributes:
{{- range $k, $v := .ResourceAttributes }}
      - action: upsert
        key: {{ $k }}
        value: {{ $v }}
{{- end }}
{{- end }}

exporters:
  otlphttp:
    endpoint: {{ .Endpoint }}
    headers:
      Authorization: "Api-Token ${env:DT_API_TOKEN}"
{{- if .HasCustomCA }}
    tls:
      ca_file: {{ .CustomCAPath }}
{{- end }}

extensions:
  health_check:
    endpoint: 0.0.0.0:13133

service:
  extensions: [health_check]
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, metric_start_time, cumulativetodelta, k8s_attributes, transform{{ if .HasResourceAttributes }}, resource/dynakube{{ end }}]
      exporters: [otlphttp]
`

var gatewayConfigTmpl = template.Must(template.New("relay").Parse(gatewayConfigTemplate))

func renderGatewayConfig(data gatewayConfigData) (string, error) {
	var buf bytes.Buffer
	if err := gatewayConfigTmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r *Reconciler) reconcileStatefulset(ctx context.Context, s *reconcileScope) error {
	if err := r.resolveImage(s); err != nil {
		return fmt.Errorf("resolve image: %w", err)
	}

	sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: s.Spec.GetStatefulSetName(), Namespace: s.Owner.Namespace}}

	err := r.createOrUpdate(ctx, s.Owner, sts, "statefulset", func() error {
		mutateStatefulSet(sts, s)

		return nil
	})
	if err != nil {
		return err
	}

	s.StatefulSet = sts

	return nil
}

func mutateStatefulSet(sts *appsv1.StatefulSet, s *reconcileScope) {
	mergeAppLabels(sts, s.AppLabels)

	sts.Spec.Template.Labels = s.Spec.Labels
	if sts.Spec.Template.Labels == nil {
		sts.Spec.Template.Labels = make(map[string]string)
	}

	maps.Copy(sts.Spec.Template.Labels, s.AppLabels.AsMap())

	sts.Spec.Template.Annotations = s.Spec.Annotations
	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = make(map[string]string)
	}

	sts.Spec.Template.Annotations[configHashAnnotation] = s.ConfigMapHash

	if s.Spec.Replicas != nil {
		// Only override when a value is set
		sts.Spec.Replicas = s.Spec.Replicas
	}

	sts.Spec.ServiceName = s.Spec.GetStatefulSetName()
	sts.Spec.PodManagementPolicy = appsv1.ParallelPodManagement
	sts.Spec.UpdateStrategy = resolveUpdateStrategy(s.Spec.UpdateStrategy)
	sts.Spec.Selector = &metav1.LabelSelector{MatchLabels: s.AppLabels.AsSelector()}
	sts.Spec.Template.Spec.ServiceAccountName = serviceAccountName
	sts.Spec.Template.Spec.AutomountServiceAccountToken = new(true)
	sts.Spec.Template.Spec.Affinity = s.Spec.Affinity
	sts.Spec.Template.Spec.NodeSelector = s.Spec.NodeSelector
	sts.Spec.Template.Spec.PriorityClassName = s.Spec.PriorityClassName
	sts.Spec.Template.Spec.Tolerations = s.Spec.Tolerations
	sts.Spec.Template.Spec.TopologySpreadConstraints = s.Spec.TopologySpreadConstraints
	sts.Spec.Template.Spec.Volumes = buildVolumes(s)
	sts.Spec.Template.Spec.Containers = []corev1.Container{buildContainer(s, getContainer(sts))}
}

// resolveUpdateStrategy defaults to plain RollingUpdate when unset. Not setting maxUnavailable
// explicitly: at least one supported apiserver version silently drops it (comes back as just
// partition: 0), so setting it unconditionally causes a permanent reconcile diff.
func resolveUpdateStrategy(strategy appsv1.StatefulSetUpdateStrategy) appsv1.StatefulSetUpdateStrategy {
	if strategy.Type != "" {
		return strategy
	}

	return appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType}
}

// getContainer returns the current (first) container of the StatefulSet, so buildContainer can
// preserve values the apiserver defaults but we don't set explicitly (e.g. ImagePullPolicy,
// probe timeouts), avoiding spurious diffs on every reconcile.
func getContainer(sts *appsv1.StatefulSet) corev1.Container {
	if len(sts.Spec.Template.Spec.Containers) > 0 {
		return sts.Spec.Template.Spec.Containers[0]
	}

	return corev1.Container{}
}

// buildContainer builds the gateway container. Minimal security context: no privileged, no
// privilege escalation, read-only root filesystem, all capabilities dropped.
func buildContainer(s *reconcileScope, current corev1.Container) corev1.Container {
	currentLivenessProbe := ptr.Deref(current.LivenessProbe, corev1.Probe{})
	currentReadinessProbe := ptr.Deref(current.ReadinessProbe, corev1.Probe{})

	imagePullPolicy := s.Spec.ImagePullPolicy
	if imagePullPolicy == "" {
		imagePullPolicy = current.ImagePullPolicy
	}

	return corev1.Container{
		Name:            "gateway",
		Image:           s.resolvedImage,
		ImagePullPolicy: imagePullPolicy,
		Command:         []string{"/dynatrace-otel-collector"},
		Args:            []string{"--config=" + configMountDir + "/" + relayConfigFile},
		Ports: []corev1.ContainerPort{
			{Name: otlpPortName, ContainerPort: otlpPort, Protocol: corev1.ProtocolTCP},
			{Name: otlpHTTPPortName, ContainerPort: otlpHTTPPort, Protocol: corev1.ProtocolTCP},
		},
		Env:          buildEnv(s),
		VolumeMounts: buildVolumeMounts(s),
		Resources:    s.Spec.Resources,
		SecurityContext: &corev1.SecurityContext{
			Privileged:               new(false),
			AllowPrivilegeEscalation: new(false),
			RunAsNonRoot:             new(true),
			ReadOnlyRootFilesystem:   new(true),
			SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{Scheme: corev1.URISchemeHTTP, Path: "/", Port: intstr.FromInt32(healthCheckPort)},
			},
			InitialDelaySeconds:           15,
			PeriodSeconds:                 20,
			TimeoutSeconds:                currentLivenessProbe.TimeoutSeconds,
			SuccessThreshold:              currentLivenessProbe.SuccessThreshold,
			FailureThreshold:              currentLivenessProbe.FailureThreshold,
			TerminationGracePeriodSeconds: currentLivenessProbe.TerminationGracePeriodSeconds,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{Scheme: corev1.URISchemeHTTP, Path: "/", Port: intstr.FromInt32(healthCheckPort)},
			},
			InitialDelaySeconds:           5,
			PeriodSeconds:                 10,
			TimeoutSeconds:                currentReadinessProbe.TimeoutSeconds,
			SuccessThreshold:              currentReadinessProbe.SuccessThreshold,
			FailureThreshold:              currentReadinessProbe.FailureThreshold,
			TerminationGracePeriodSeconds: currentReadinessProbe.TerminationGracePeriodSeconds,
		},
		TerminationMessagePath:   current.TerminationMessagePath,
		TerminationMessagePolicy: current.TerminationMessagePolicy,
	}
}

// fieldRef sets APIVersion explicitly to "v1" — the value the apiserver defaults it to when
// omitted, so a freshly-built selector compares equal to the stored, already-defaulted one.
func fieldRef(fieldPath string) *corev1.ObjectFieldSelector {
	return &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: fieldPath}
}

// fieldRefEnv builds a Downward-API-sourced env var: k8s populates the value at pod start, we
// just declare which field goes where.
func fieldRefEnv(name, fieldPath string) corev1.EnvVar {
	return corev1.EnvVar{Name: name, ValueFrom: &corev1.EnvVarSource{FieldRef: fieldRef(fieldPath)}}
}

// buildEnv builds the gateway container's environment variables.
func buildEnv(s *reconcileScope) []corev1.EnvVar {
	dk := s.DynaKube

	envs := []corev1.EnvVar{
		fieldRefEnv("MY_POD_IP", "status.podIP"),
		{Name: "DT_API_TOKEN", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: dk.Tokens()},
			Key:                  token.APIKey,
		}}},
		{Name: "K8S_CLUSTER_NAME", Value: dk.Status.KubernetesClusterName},
		{Name: "DT_ENTITY_KUBERNETES_CLUSTER", Value: dk.Status.KubernetesClusterMEID},
		fieldRefEnv("K8S_NODE_NAME", "spec.nodeName"),
		fieldRefEnv("K8S_POD_NAME", "metadata.name"),
		fieldRefEnv("K8S_NAMESPACE_NAME", "metadata.namespace"),
		fieldRefEnv("K8S_POD_UID", "metadata.uid"),
	}

	if memLimitEnv, ok := goMemLimitEnv(s.Spec.Resources); ok {
		envs = append(envs, memLimitEnv)
	}

	if dk.HasProxy() {
		envs = append(envs,
			proxyEnv("HTTPS_PROXY", dk.Spec.Proxy),
			proxyEnv("HTTP_PROXY", dk.Spec.Proxy),
			corev1.EnvVar{Name: "NO_PROXY", Value: noProxyValue(dk)},
		)
	}

	return envs
}

// goMemLimitEnv sets GOMEMLIMIT to 90% of the configured memory limit; omitted when no limit is set.
func goMemLimitEnv(resources corev1.ResourceRequirements) (corev1.EnvVar, bool) {
	limit, ok := resources.Limits[corev1.ResourceMemory]
	if !ok {
		return corev1.EnvVar{}, false
	}

	bytes := int64(float64(limit.Value()) * 0.9)

	return corev1.EnvVar{Name: "GOMEMLIMIT", Value: strconv.FormatInt(bytes, 10)}, true
}

func proxyEnv(name string, src *value.Source) corev1.EnvVar {
	if src.ValueFrom != "" {
		return corev1.EnvVar{
			Name: name,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: src.ValueFrom},
					Key:                  dynakube.ProxyKey,
				},
			},
		}
	}

	return corev1.EnvVar{Name: name, Value: src.Value}
}

// noProxyValue mirrors the NO_PROXY convention used by the DynaKube's own otelc component:
// cluster-internal traffic plus the ActiveGate service, when routed through one.
func noProxyValue(dk *dynakube.DynaKube) string {
	values := []string{"$(KUBERNETES_SERVICE_HOST)", "kubernetes.default"}

	if dk.ActiveGate().IsEnabled() {
		values = append(values, capability.BuildServiceName(dk.Name)+"."+dk.Namespace)
	}

	return strings.Join(values, ",")
}

// buildVolumes always mounts the rendered relay.yaml ConfigMap, plus whichever CA the exporter
// needs to trust (AG's cert when routed through it, the user-provided trusted CA otherwise).
func buildVolumes(s *reconcileScope) []corev1.Volume {
	dk := s.DynaKube

	// DefaultMode is set explicitly everywhere below (matching what the apiserver defaults it
	// to) so a freshly-built Volume compares equal to the stored, already-defaulted one on the
	// next reconcile — otherwise every reconcile sees a nil-vs-0644 diff and issues a spurious
	// Update.
	defaultMode := new(int32(0o644))

	volumes := []corev1.Volume{
		{
			Name: configVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: s.Spec.GetStatefulSetName()},
					Items:                []corev1.KeyToPath{{Key: gatewayConfigKey, Path: relayConfigFile}},
					DefaultMode:          defaultMode,
				},
			},
		},
	}

	switch {
	case dk.IsAGCertificateNeeded():
		volumes = append(volumes, corev1.Volume{
			Name: agCertVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  dk.ActiveGate().GetTLSSecretName(),
					Items:       []corev1.KeyToPath{{Key: dynakube.ServerCertKey, Path: activeGateCertFile}},
					DefaultMode: defaultMode,
				},
			},
		})
	case dk.IsCACertificateNeeded():
		volumes = append(volumes, corev1.Volume{
			Name: cacertsVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: dk.Spec.TrustedCAs},
					Items:                []corev1.KeyToPath{{Key: "certs", Path: trustedCAFile}},
					DefaultMode:          defaultMode,
				},
			},
		})
	}

	return volumes
}

func buildVolumeMounts(s *reconcileScope) []corev1.VolumeMount {
	dk := s.DynaKube

	mounts := []corev1.VolumeMount{
		{Name: configVolumeName, MountPath: configMountDir, ReadOnly: true},
	}

	switch {
	case dk.IsAGCertificateNeeded():
		mounts = append(mounts, corev1.VolumeMount{Name: agCertVolumeName, MountPath: activeGateCertVolumeMountPath, ReadOnly: true})
	case dk.IsCACertificateNeeded():
		mounts = append(mounts, corev1.VolumeMount{Name: cacertsVolumeName, MountPath: trustedCAVolumeMountPath, ReadOnly: true})
	}

	return mounts
}

func (r *Reconciler) reconcileService(ctx context.Context, s *reconcileScope) error {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: s.Spec.GetStatefulSetName(), Namespace: s.Owner.Namespace}}

	return r.createOrUpdate(ctx, s.Owner, svc, "service", func() error {
		mergeAppLabels(svc, s.AppLabels)

		svc.Spec.Selector = s.AppLabels.AsSelector()
		svc.Spec.Ports = []corev1.ServicePort{
			{
				Name:        otlpPortName,
				Protocol:    corev1.ProtocolTCP,
				Port:        otlpPort,
				AppProtocol: new("grpc"),
				TargetPort:  intstr.FromString(otlpPortName),
			},
			{
				Name:       otlpHTTPPortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       otlpHTTPPort,
				TargetPort: intstr.FromString(otlpHTTPPortName),
			},
		}

		return nil
	})
}
