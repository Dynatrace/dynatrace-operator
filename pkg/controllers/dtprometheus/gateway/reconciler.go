// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dtprometheus/condition"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// gatewayConfigKey is the ConfigMap data key holding the rendered OTel Collector config,
	// mounted as relay.yaml (see --config=/conf/relay.yaml on the container).
	gatewayConfigKey     = "relay"
	configHashAnnotation = "internal.operator.dynatrace.com/gateway-config-hash"

	trustedCAVolumeMountPath = "/tls/custom/cacerts"
	trustedCAFile            = "rootca.pem"

	otlpPortName = "otlp"
	otlpPort     = 4317

	healthCheckPort = 13133

	serviceAccountName = "dynatrace-prometheus-gateway"

	// otelCollectorNonRootUser is the "nonroot" UID/GID the gateway image runs as.
	otelCollectorNonRootUser = 10001

	configVolumeName  = "opentelemetry-collector-configmap"
	configMountDir    = "/conf"
	relayConfigFile   = "relay.yaml"
	cacertsVolumeName = "cacerts"

	// Mounted as a directory, not subPath: subPath mounts don't receive live Secret
	// updates, which would break token rotation.
	tokenVolumeName = "dt-token"
	tokenMountPath  = consts.DTComponentsSecretsRootDir + "/tokens"
	tokenFileName   = "data-ingest-token"
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

	condition.Set(&scope.Owner.Status.Conditions, dtprometheus.GatewayAvailable, "gateway",
		func() bool { return k8sstatefulset.IsRolloutComplete(scope.StatefulSet) }, err)

	return err
}

// resolveImage uses the explicit image from .spec.gateway.image when set, otherwise resolves
// the latest gateway image from the fleet management API.
func (r *Reconciler) resolveImage(ctx context.Context, s *reconcileScope) error {
	imageURI := s.Spec.Image

	if imageURI == "" {
		var err error

		imageURI, err = registry.ResolveImage(ctx, s.ImageClient, s.Owner.Spec.PublicRegistryOverride, image.Gateway)
		if err != nil {
			return err
		}
	}

	s.Owner.Status.Gateway.ResolvedImage = imageURI

	return nil
}

func (r *Reconciler) reconcileConfigMap(ctx context.Context, s *reconcileScope) error {
	name := s.Spec.GetStatefulSetName()

	rendered, err := renderGatewayConfig(buildGatewayConfigData(s.DynaKube))
	if err != nil {
		return fmt.Errorf("render gateway config: %w", err)
	}

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.Owner.Namespace}}

	err = k8sobject.RetryCreateOrUpdate(ctx, r, cm, func() error {
		s.AppLabels.MergeInto(cm)
		cm.Data = map[string]string{gatewayConfigKey: rendered}

		return controllerutil.SetControllerReference(s.Owner, cm, r.Scheme())
	})
	if err != nil {
		return err
	}

	checksum := sha256.Sum256([]byte(rendered))
	s.ConfigMapHash = hex.EncodeToString(checksum[:])

	return nil
}

type gatewayConfigData struct {
	Endpoint           string
	CustomCAPath       string
	ResourceAttributes map[string]string
}

// buildGatewayConfigData resolves the DynaKube-derived inputs to the relay.yaml template.
func buildGatewayConfigData(dk *dynakube.DynaKube) gatewayConfigData {
	data := gatewayConfigData{
		Endpoint:           dk.APIURL() + "/v2/otlp",
		ResourceAttributes: dk.GetResourceAttributes(),
	}

	if dk.Spec.TrustedCAs != "" {
		data.CustomCAPath = filepath.Join(trustedCAVolumeMountPath, trustedCAFile)
	}

	return data
}

func renderGatewayConfig(data gatewayConfigData) (string, error) {
	b, err := buildGatewayOTelConfig(data).Marshal()

	return string(b), err
}

func (r *Reconciler) reconcileStatefulset(ctx context.Context, s *reconcileScope) error {
	if err := r.resolveImage(ctx, s); err != nil {
		return fmt.Errorf("resolve image: %w", err)
	}

	sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: s.Spec.GetStatefulSetName(), Namespace: s.Owner.Namespace}}

	err := k8sobject.RetryCreateOrUpdate(ctx, r, sts, func() error {
		mutateStatefulSet(sts, s)

		return controllerutil.SetControllerReference(s.Owner, sts, r.Scheme())
	})
	if err != nil {
		return err
	}

	s.StatefulSet = sts

	return nil
}

func mutateStatefulSet(sts *appsv1.StatefulSet, s *reconcileScope) {
	s.AppLabels.MergeInto(sts)

	sts.Spec.Template.Labels = maps.Clone(s.Spec.Labels)
	if sts.Spec.Template.Labels == nil {
		sts.Spec.Template.Labels = make(map[string]string)
	}

	maps.Copy(sts.Spec.Template.Labels, s.AppLabels.AsMap())

	sts.Spec.Template.Annotations = maps.Clone(s.Spec.Annotations)
	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = make(map[string]string)
	}

	sts.Spec.Template.Annotations[configHashAnnotation] = s.ConfigMapHash

	if s.Spec.Replicas != nil {
		sts.Spec.Replicas = s.Spec.Replicas
	}

	sts.Spec.ServiceName = s.Spec.GetStatefulSetName()

	sts.Spec.PodManagementPolicy = appsv1.ParallelPodManagement

	sts.Spec.Selector = &metav1.LabelSelector{MatchLabels: s.AppLabels.AsSelector()}
	sts.Spec.Template.Spec.ServiceAccountName = serviceAccountName
	sts.Spec.Template.Spec.AutomountServiceAccountToken = new(true)
	// fsGroup lets the container (RunAsGroup nonRootUser) read the token volume via the group
	// bit, so the file doesn't need to be world-readable.
	sts.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{FSGroup: new(int64(otelCollectorNonRootUser))}
	sts.Spec.Template.Spec.Affinity = s.Spec.Affinity
	sts.Spec.Template.Spec.NodeSelector = s.Spec.NodeSelector
	sts.Spec.Template.Spec.PriorityClassName = s.Spec.PriorityClassName
	sts.Spec.Template.Spec.Tolerations = s.Spec.Tolerations
	sts.Spec.Template.Spec.TopologySpreadConstraints = s.Spec.TopologySpreadConstraints
	sts.Spec.Template.Spec.Volumes = buildVolumes(s)
	// The stored container is passed in so buildContainer can preserve apiserver-defaulted
	// fields (e.g. ImagePullPolicy, probe timeouts) and avoid spurious diffs.
	sts.Spec.Template.Spec.Containers = []corev1.Container{
		buildContainer(s, k8scontainer.GetFirstInPodSpec(&sts.Spec.Template.Spec)),
	}
}

func buildContainer(s *reconcileScope, current corev1.Container) corev1.Container {
	currentLivenessProbe := ptr.Deref(current.LivenessProbe, corev1.Probe{})
	currentReadinessProbe := ptr.Deref(current.ReadinessProbe, corev1.Probe{})

	imagePullPolicy := s.Spec.ImagePullPolicy
	if imagePullPolicy == "" {
		imagePullPolicy = current.ImagePullPolicy
	}

	return corev1.Container{
		Name:            "gateway",
		Image:           s.Owner.Status.Gateway.ResolvedImage,
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
			RunAsUser:                new(int64(otelCollectorNonRootUser)),
			RunAsGroup:               new(int64(otelCollectorNonRootUser)),
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

func buildEnv(s *reconcileScope) []corev1.EnvVar {
	dk := s.DynaKube

	envs := k8senv.AppendGoMemoryLimit([]corev1.EnvVar{
		{
			Name: "MY_POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				// APIVersion must be set explicitly: the API server defaults it to "v1" on
				// storage, so omitting it here would cause a reconcile diff on every iteration.
				FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "status.podIP"},
			},
		},
	}, s.Spec.Resources)

	if dk.HasProxy() {
		envs = append(
			envs,
			proxyEnv("HTTPS_PROXY", dk.Spec.Proxy),
			proxyEnv("HTTP_PROXY", dk.Spec.Proxy),
			corev1.EnvVar{Name: "NO_PROXY", Value: noProxyValue(dk)},
		)
	}

	return envs
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

func noProxyValue(dk *dynakube.DynaKube) string {
	values := []string{"$(KUBERNETES_SERVICE_HOST)", "kubernetes.default"}

	if dk.ActiveGate().IsEnabled() {
		values = append(values, capability.BuildServiceName(dk.Name)+"."+dk.Namespace)
	}

	return strings.Join(values, ",")
}

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
		{
			Name: tokenVolumeName,
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					// Group-readable, not world-readable: the pod's fsGroup grants read access to
					// the container without exposing the file to any other UID.
					DefaultMode: new(int32(0o440)),
					Sources: []corev1.VolumeProjection{
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{Name: dk.Tokens()},
								Items: []corev1.KeyToPath{
									{Key: token.DataIngestKey, Path: tokenFileName},
								},
							},
						},
					},
				},
			},
		},
	}

	if dk.Spec.TrustedCAs != "" {
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
		{Name: tokenVolumeName, MountPath: tokenMountPath, ReadOnly: true},
	}

	if dk.Spec.TrustedCAs != "" {
		mounts = append(mounts, corev1.VolumeMount{Name: cacertsVolumeName, MountPath: trustedCAVolumeMountPath, ReadOnly: true})
	}

	return mounts
}

func (r *Reconciler) reconcileService(ctx context.Context, s *reconcileScope) error {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: s.Spec.GetStatefulSetName(), Namespace: s.Owner.Namespace}}

	return k8sobject.RetryCreateOrUpdate(ctx, r, svc, func() error {
		s.AppLabels.MergeInto(svc)

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

		return controllerutil.SetControllerReference(s.Owner, svc, r.Scheme())
	})
}
