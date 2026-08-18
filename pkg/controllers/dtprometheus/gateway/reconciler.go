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
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/registry"
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

	trustedCAVolumeMountPath = "/tls/custom/cacerts"
	trustedCAFile            = "rootca.pem"

	otlpPortName = "otlp"
	otlpPort     = 4317

	healthCheckPort = 13133

	serviceAccountName = "dynatrace-prometheus-gateway"

	configVolumeName  = "opentelemetry-collector-configmap"
	configMountDir    = "/conf"
	relayConfigFile   = "relay.yaml"
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

// resolveImage uses the explicit image from .spec.gateway.image when set, otherwise resolves
// the latest gateway image from the fleet management API. The resolved image is also stamped
// onto .status.gateway.image, for informational purposes only.
func (r *Reconciler) resolveImage(ctx context.Context, s *reconcileScope) error {
	imageURI := s.Spec.Image

	if imageURI == "" {
		var err error

		imageURI, err = registry.ResolveImage(ctx, s.ImageClient, s.Owner.Spec.PublicRegistryOverride, image.Gateway)
		if err != nil {
			return err
		}
	}

	s.resolvedImage = imageURI
	s.Owner.Status.Gateway.ResolvedImage = imageURI

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
// Routing decision is based specifically on the routing ActiveGate capability, not just "any AG
// enabled" — customCA/proxy only apply in the no-routing-AG (direct-to-tenant) case, per the story.
func buildGatewayConfigData(dk *dynakube.DynaKube) (gatewayConfigData, error) {
	endpoint, err := gatewayEndpoint(dk)
	if err != nil {
		return gatewayConfigData{}, err
	}

	data := gatewayConfigData{Endpoint: endpoint}

	if !dk.ActiveGate().IsRoutingEnabled() && dk.Spec.TrustedCAs != "" {
		data.HasCustomCA = true
		data.CustomCAPath = trustedCAVolumeMountPath + "/" + trustedCAFile
	}

	if attrs := dk.GetResourceAttributes(); len(attrs) > 0 {
		data.ResourceAttributes = attrs
		data.HasResourceAttributes = true
	}

	return data, nil
}

// gatewayEndpoint routes through the routing ActiveGate when configured, otherwise sends
// directly to the tenant. Both paths are always HTTPS — "plain HTTP" in the story refers to the
// gateway's own inbound OTLP receiver, not this egress.
func gatewayEndpoint(dk *dynakube.DynaKube) (string, error) {
	if !dk.ActiveGate().IsRoutingEnabled() {
		return dk.APIURL() + "/v2/otlp", nil
	}

	tenantUUID, err := dk.TenantUUID()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s.%s/e/%s/api/v2/otlp", capability.BuildServiceName(dk.Name), dk.Namespace, tenantUUID), nil
}

// gatewayConfigTemplate renders the OTel Collector relay.yaml for the gateway StatefulSet.
// Derived from the POC (tier2-gateway.rendered.yaml), adapted to meet AC:
//   - No TLS on the inbound receiver (plain HTTP story; TLS in a later story).
//   - No self-monitoring telemetry section.
//   - Cluster-name and dt.entity.kubernetes_cluster enrichment omitted (AC: "not enriched with
//     cluster name or kubernetes meid").
//   - customCA/proxy applied only when no routing ActiveGate is configured.
const gatewayConfigTemplate = `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: ${env:MY_POD_IP}:4317

processors:
  memory_limiter:
    check_interval: 1s
    limit_percentage: 95
    spike_limit_percentage: 5
  metric_start_time: {}
  cumulativetodelta:
    initial_value: drop
    max_staleness: 10m
  k8s_attributes:
    extract:
      annotations:
      - from: pod
        key_regex: metadata.dynatrace.com/(.*)
        tag_name: $$1
      - from: pod
        key: metadata.dynatrace.com
        tag_name: metadata.dynatrace.com
      metadata:
      - k8s.pod.name
      - k8s.pod.uid
      - k8s.pod.ip
      - k8s.deployment.name
      - k8s.replicaset.name
      - k8s.statefulset.name
      - k8s.daemonset.name
      - k8s.job.name
      - k8s.cronjob.name
      - k8s.namespace.name
      - k8s.node.name
      - k8s.cluster.uid
      - k8s.container.name
      - k8s.deployment.uid
      - k8s.replicaset.uid
      - k8s.statefulset.uid
      - k8s.daemonset.uid
      - k8s.job.uid
      - k8s.cronjob.uid
    pod_association:
    - sources:
      - from: resource_attribute
        name: server.address
    - sources:
      - from: resource_attribute
        name: k8s.pod.name
      - from: resource_attribute
        name: k8s.namespace.name
    - sources:
      - from: resource_attribute
        name: k8s.pod.ip
    - sources:
      - from: resource_attribute
        name: k8s.pod.uid
    - sources:
      - from: connection
  transform:
    metric_statements:
    - context: datapoint
      statements:
      - set(attributes["http.request.method"], attributes["http_request_method"]) where attributes["http_request_method"] != nil
      - delete_key(attributes, "http_request_method")
      - set(attributes["http.response.status_code"], attributes["http_response_status_code"]) where attributes["http_response_status_code"] != nil
      - delete_key(attributes, "http_response_status_code")
      - set(attributes["network.protocol.name"], attributes["network_protocol_name"]) where attributes["network_protocol_name"] != nil
      - delete_key(attributes, "network_protocol_name")
      - set(attributes["network.protocol.version"], attributes["network_protocol_version"]) where attributes["network_protocol_version"] != nil
      - delete_key(attributes, "network_protocol_version")
      - set(attributes["rpc.method"], attributes["rpc_method"]) where attributes["rpc_method"] != nil
      - delete_key(attributes, "rpc_method")
      - set(attributes["rpc.response.status_code"], attributes["rpc_response_status_code"]) where attributes["rpc_response_status_code"] != nil
      - delete_key(attributes, "rpc_response_status_code")
      - set(attributes["rpc.system.name"], attributes["rpc_system_name"]) where attributes["rpc_system_name"] != nil
      - delete_key(attributes, "rpc_system_name")
      - set(attributes["server.address"], attributes["server_address"]) where attributes["server_address"] != nil
      - delete_key(attributes, "server_address")
      - set(attributes["server.port"], attributes["server_port"]) where attributes["server_port"] != nil
      - delete_key(attributes, "server_port")
      - set(attributes["url.scheme"], attributes["url_scheme"]) where attributes["url_scheme"] != nil
      - delete_key(attributes, "url_scheme")
    - context: resource
      statements:
      - set(attributes["k8s.workload.name"], attributes["k8s.statefulset.name"]) where IsString(attributes["k8s.statefulset.name"])
      - set(attributes["k8s.workload.name"], attributes["k8s.replicaset.name"]) where IsString(attributes["k8s.replicaset.name"])
      - set(attributes["k8s.workload.name"], attributes["k8s.job.name"]) where IsString(attributes["k8s.job.name"])
      - set(attributes["k8s.workload.name"], attributes["k8s.deployment.name"]) where IsString(attributes["k8s.deployment.name"])
      - set(attributes["k8s.workload.name"], attributes["k8s.daemonset.name"]) where IsString(attributes["k8s.daemonset.name"])
      - set(attributes["k8s.workload.name"], attributes["k8s.cronjob.name"]) where IsString(attributes["k8s.cronjob.name"])
      - set(attributes["k8s.workload.kind"], "statefulset") where IsString(attributes["k8s.statefulset.name"])
      - set(attributes["k8s.workload.kind"], "replicaset") where IsString(attributes["k8s.replicaset.name"])
      - set(attributes["k8s.workload.kind"], "job") where IsString(attributes["k8s.job.name"])
      - set(attributes["k8s.workload.kind"], "deployment") where IsString(attributes["k8s.deployment.name"])
      - set(attributes["k8s.workload.kind"], "daemonset") where IsString(attributes["k8s.daemonset.name"])
      - set(attributes["k8s.workload.kind"], "cronjob") where IsString(attributes["k8s.cronjob.name"])
      - set(attributes["k8s.workload.uid"], attributes["k8s.statefulset.uid"]) where IsString(attributes["k8s.statefulset.uid"])
      - set(attributes["k8s.workload.uid"], attributes["k8s.replicaset.uid"]) where IsString(attributes["k8s.replicaset.uid"])
      - set(attributes["k8s.workload.uid"], attributes["k8s.job.uid"]) where IsString(attributes["k8s.job.uid"])
      - set(attributes["k8s.workload.uid"], attributes["k8s.deployment.uid"]) where IsString(attributes["k8s.deployment.uid"])
      - set(attributes["k8s.workload.uid"], attributes["k8s.daemonset.uid"]) where IsString(attributes["k8s.daemonset.uid"])
      - set(attributes["k8s.workload.uid"], attributes["k8s.cronjob.uid"]) where IsString(attributes["k8s.cronjob.uid"])
      - delete_key(attributes, "k8s.statefulset.name")
      - delete_key(attributes, "k8s.replicaset.name")
      - delete_key(attributes, "k8s.job.name")
      - delete_key(attributes, "k8s.deployment.name")
      - delete_key(attributes, "k8s.daemonset.name")
      - delete_key(attributes, "k8s.cronjob.name")
      - delete_key(attributes, "k8s.statefulset.uid")
      - delete_key(attributes, "k8s.replicaset.uid")
      - delete_key(attributes, "k8s.deployment.uid")
      - delete_key(attributes, "k8s.daemonset.uid")
      - delete_key(attributes, "k8s.job.uid")
      - delete_key(attributes, "k8s.cronjob.uid")
    - context: resource
      statements:
      - delete_key(attributes, "processor")
      - delete_key(attributes, "otel.signal")
      - delete_key(attributes, "otel.scope.name")
      - delete_key(attributes, "otel.scope.version")
    - context: resource
      statements:
      - merge_maps(attributes, ParseJSON(attributes["metadata.dynatrace.com"]), "upsert") where IsMatch(attributes["metadata.dynatrace.com"], "^\\{")
      - delete_key(attributes, "metadata.dynatrace.com")
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
    sending_queue:
      batch:
        flush_timeout: 10s
        max_size: 5000
        min_size: 500
{{- if .HasCustomCA }}
    tls:
      ca_file: {{ .CustomCAPath }}
{{- end }}

extensions:
  health_check:
    endpoint: ${env:MY_POD_IP}:13133

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
	if err := r.resolveImage(ctx, s); err != nil {
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

// resolveUpdateStrategy defaults to RollingUpdate/partition:0 when unset, per the story. Not
// setting maxUnavailable explicitly: at least one supported apiserver version silently drops it
// (comes back as just partition: 0), so setting it unconditionally causes a permanent reconcile diff.
func resolveUpdateStrategy(strategy appsv1.StatefulSetUpdateStrategy) appsv1.StatefulSetUpdateStrategy {
	if strategy.Type != "" {
		return strategy
	}

	return appsv1.StatefulSetUpdateStrategy{
		Type:          appsv1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: new(int32(0))},
	}
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
		},
		Env:          buildEnv(s),
		VolumeMounts: buildVolumeMounts(s),
		Resources:    s.Spec.Resources,
		SecurityContext: &corev1.SecurityContext{
			Privileged:               new(false),
			AllowPrivilegeEscalation: new(false),
			RunAsNonRoot:             new(true),
			RunAsUser:                new(int64(65532)),
			RunAsGroup:               new(int64(65532)),
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

// buildEnv builds the gateway container's environment variables.
func buildEnv(s *reconcileScope) []corev1.EnvVar {
	dk := s.DynaKube

	envs := []corev1.EnvVar{
		// MY_POD_IP is used in the relay.yaml config to bind the OTLP receiver and health-check
		// endpoint to the pod IP rather than 0.0.0.0, matching the POC pattern.
		{
			Name: "MY_POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				// APIVersion must be set explicitly: the API server defaults it to "v1" on
				// storage, so omitting it here would cause a reconcile diff on every iteration.
				FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "status.podIP"},
			},
		},
		{Name: "DT_API_TOKEN", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: dk.Tokens()},
			Key:                  token.APIKey,
		}}},
	}

	if memLimitEnv, ok := goMemLimitEnv(s.Spec.Resources); ok {
		envs = append(envs, memLimitEnv)
	}

	// Proxy only applies when sending directly to the tenant: a routing ActiveGate handles its
	// own egress, the gateway never talks to the internet directly in that case.
	if !dk.ActiveGate().IsRoutingEnabled() && dk.HasProxy() {
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

// buildVolumes always mounts the rendered relay.yaml ConfigMap, plus the user-provided trusted CA
// when sending directly to the tenant (no routing ActiveGate configured).
func buildVolumes(s *reconcileScope) []corev1.Volume {
	dk := s.DynaKube

	// DefaultMode is set explicitly (matching what the apiserver defaults it to) so a freshly-built
	// Volume compares equal to the stored, already-defaulted one on the next reconcile — otherwise
	// every reconcile sees a nil-vs-0644 diff and issues a spurious Update.
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

	if !dk.ActiveGate().IsRoutingEnabled() && dk.Spec.TrustedCAs != "" {
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

	if !dk.ActiveGate().IsRoutingEnabled() && dk.Spec.TrustedCAs != "" {
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
		}

		return nil
	})
}
