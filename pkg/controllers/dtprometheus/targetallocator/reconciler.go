// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sdeployment"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	insecurePortName = "http-port"
	securePortName   = "https-port"

	configVolume = "config"
	configFile   = "targetallocator.yaml"

	serviceAccount = "dynatrace-target-allocator"
)

type Reconciler struct {
	client.Client
}

// Config is the subset of configurations that can be configured by the operator.
//
// https://github.com/open-telemetry/opentelemetry-operator/blob/v0.157.0/cmd/otel-allocator/internal/config/config.go
type Config struct {
	ListenAddr         string                `json:"listen_addr"`
	AllocationStrategy string                `json:"allocation_strategy"`
	CollectorNamespace string                `json:"collector_namespace"`
	CollectorSelector  *metav1.LabelSelector `json:"collector_selector,omitempty"`
	FilterStrategy     string                `json:"filter_strategy"`
	HTTPS              *HTTPSConfig          `json:"https,omitempty"`
	PrometheusCR       ScrapeConfig          `json:"prometheus_cr"`
}

type HTTPSConfig struct {
	Enabled         bool   `json:"enabled"`
	ListenAddr      string `json:"listen_addr"`
	TLSCertFilePath string `json:"tls_cert_file_path"`
	TLSKeyFilePath  string `json:"tls_keyt_file_path"`
	CAFilePath      string `json:"ca_cert_file_path"`
}

type ScrapeConfig struct {
	Enabled                         bool                  `json:"enabled"`
	ScrapeInterval                  metav1.Duration       `json:"scrape_interval"`
	PodMonitorSelector              *metav1.LabelSelector `json:"pod_monitor_selector,omitempty"`
	PodMonitorNamespaceSelector     *metav1.LabelSelector `json:"pod_monitor_namespace_selector,omitempty"`
	ServiceMonitorSelector          *metav1.LabelSelector `json:"service_monitor_selector,omitempty"`
	ServiceMonitorNamespaceSelector *metav1.LabelSelector `json:"service_monitor_namespace_selector,omitempty"`
	ScrapeConfigSelector            *metav1.LabelSelector `json:"scrape_config_selector,omitempty"`
	ScrapeConfigNamespaceSelector   *metav1.LabelSelector `json:"scrape_config_namespace_selector,omitempty"`
	ProbeSelector                   *metav1.LabelSelector `json:"probe_selector,omitempty"`
	ProbeNamespaceSelector          *metav1.LabelSelector `json:"probe_namespace_selector,omitempty"`
}

type reconcileScope struct {
	// Required for reconcile
	Owner       *dtprometheus.DTPrometheus
	DynaKube    *dynakube.DynaKube
	Spec        *dtprometheus.TargetAllocator
	AppLabels   *k8slabel.Labels
	ImageClient image.Client
	// Computed during reconcile
	ConfigMapHash string
	Deployment    *appsv1.Deployment
}

func (r *Reconciler) Reconcile(ctx context.Context, dtp *dtprometheus.DTPrometheus, dk *dynakube.DynaKube, imageClient image.Client) error {
	ctx, _ = logd.NewFromContext(ctx, "targetallocator")

	scope := &reconcileScope{
		Owner:       dtp,
		DynaKube:    dk,
		Spec:        dtp.TargetAllocator(),
		AppLabels:   k8slabel.OTelTargetAllocator(),
		ImageClient: imageClient,
	}

	var err error

	for _, f := range []func(context.Context, *reconcileScope) error{
		r.reconcileConfigMap,
		r.reconcileDeployment,
		r.reconcileService,
	} {
		if err = f(ctx, scope); err != nil {
			break
		}
	}

	r.reconcileCondition(scope, err)

	return err
}

func (r *Reconciler) reconcileCondition(s *reconcileScope, err error) {
	condition := metav1.Condition{
		Type: dtprometheus.TargetAllocatorAvailable,
	}

	switch {
	case err != nil:
		condition.Status = metav1.ConditionFalse
		condition.Reason = status.ReasonError
		condition.Message = safeUnwrap(err).Error()
	case k8sdeployment.IsRolloutComplete(s.Deployment):
		condition.Status = metav1.ConditionTrue
		condition.Reason = status.ReasonAvailable
		condition.Message = "target allocator is ready"
	default:
		condition.Status = metav1.ConditionFalse
		condition.Reason = status.ReasonReconciling
		condition.Message = "target allocator is pending"
	}

	_ = meta.SetStatusCondition(&s.Owner.Status.Conditions, condition)
}

func (r *Reconciler) reconcileConfigMap(ctx context.Context, s *reconcileScope) error {
	log := logd.FromContext(ctx)
	log.Debug("reconciling configmap")

	cfg := Config{
		ListenAddr:         ":8080",
		AllocationStrategy: "consistent-hashing",
		CollectorNamespace: s.Owner.Namespace,
		CollectorSelector: &metav1.LabelSelector{
			MatchLabels: k8slabel.OTelScraper().AsSelector(),
		},
		FilterStrategy: "relabel-config",
		HTTPS: &HTTPSConfig{
			Enabled:    false,
			ListenAddr: ":8443",
			// TODO: add cert mounts
		},
		PrometheusCR: ScrapeConfig{
			Enabled:                         true,
			ScrapeInterval:                  s.Spec.ScrapeInterval,
			PodMonitorSelector:              s.Spec.ScrapeCRSelector,
			PodMonitorNamespaceSelector:     s.Spec.ScrapeCRNamespaceSelector,
			ServiceMonitorSelector:          s.Spec.ScrapeCRSelector,
			ServiceMonitorNamespaceSelector: s.Spec.ScrapeCRNamespaceSelector,
			ScrapeConfigSelector:            s.Spec.ScrapeCRSelector,
			ScrapeConfigNamespaceSelector:   s.Spec.ScrapeCRNamespaceSelector,
			ProbeSelector:                   s.Spec.ScrapeCRSelector,
			ProbeNamespaceSelector:          s.Spec.ScrapeCRNamespaceSelector,
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: s.Spec.GetDeploymentName(), Namespace: s.Owner.Namespace}}

	result, err := k8sobject.RetryCreateOrUpdate(ctx, r, cm, func() error {
		if cm.Labels == nil {
			cm.Labels = make(map[string]string)
		}

		maps.Copy(cm.Labels, s.AppLabels.AsMap())

		cm.Data = map[string]string{configFile: string(data)}

		return controllerutil.SetControllerReference(s.Owner, cm, r.Scheme())
	})
	if err != nil {
		return fmt.Errorf("reconcile configmap: %w", err)
	}

	checksum := sha256.Sum256(data)
	s.ConfigMapHash = hex.EncodeToString(checksum[:])

	switch result {
	case controllerutil.OperationResultCreated:
		log.Info("created configmap")
	case controllerutil.OperationResultUpdated:
		log.Info("updated configmap")
	}

	return nil
}

func (r *Reconciler) reconcileDeployment(ctx context.Context, s *reconcileScope) error {
	log := logd.FromContext(ctx)
	log.Debug("reconciling deployment")

	if s.Spec.Image == "" {
		// TODO: fix this once the target allocator images are available
		// imageURI, err := registry.ResolveImage(ctx, s.Image, s.Owner.Spec.PublicRegistryOverride, image.TargetAllocator)
		return errors.New("missing image")
	}

	deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: s.Spec.GetDeploymentName(), Namespace: s.Owner.Namespace}}

	result, err := k8sobject.RetryCreateOrUpdate(ctx, r, deploy, func() error {
		mutateDeployment(deploy, s)

		return controllerutil.SetControllerReference(s.Owner, deploy, r.Scheme())
	})
	if err != nil {
		return fmt.Errorf("reconcile deployment: %w", err)
	}

	switch result {
	case controllerutil.OperationResultCreated:
		log.Info("created deployment")
	case controllerutil.OperationResultUpdated:
		log.Info("updated deployment")
	}

	s.Deployment = deploy

	return nil
}

func (r *Reconciler) reconcileService(ctx context.Context, s *reconcileScope) error {
	log := logd.FromContext(ctx)
	log.Debug("reconciling service")

	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: s.Spec.GetDeploymentName(), Namespace: s.Owner.Namespace}}

	result, err := k8sobject.RetryCreateOrUpdate(ctx, r, svc, func() error {
		if svc.Labels == nil {
			svc.Labels = make(map[string]string)
		}

		maps.Copy(svc.Labels, s.AppLabels.AsMap())

		svc.Spec.Selector = s.AppLabels.AsSelector()
		svc.Spec.Ports = []corev1.ServicePort{
			{
				Name:       securePortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       443,
				TargetPort: intstr.FromString(securePortName),
			},
			{
				Name:       insecurePortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       80,
				TargetPort: intstr.FromString(insecurePortName),
			},
		}

		return controllerutil.SetControllerReference(s.Owner, svc, r.Scheme())
	})
	if err != nil {
		return fmt.Errorf("reconcile service: %w", err)
	}

	switch result {
	case controllerutil.OperationResultCreated:
		log.Info("created service")
	case controllerutil.OperationResultUpdated:
		log.Info("updated service")
	}

	return nil
}

func mutateDeployment(deploy *appsv1.Deployment, s *reconcileScope) {
	if deploy.Labels == nil {
		deploy.Labels = make(map[string]string)
	}

	maps.Copy(deploy.Labels, s.AppLabels.AsMap())

	deploy.Spec.Template.Labels = s.Spec.Labels
	if s.Spec.Labels == nil {
		deploy.Spec.Template.Labels = make(map[string]string)
	}

	maps.Copy(deploy.Spec.Template.Labels, s.AppLabels.AsMap())

	deploy.Spec.Template.Annotations = s.Spec.Annotations
	if deploy.Spec.Template.Annotations == nil {
		deploy.Spec.Template.Annotations = make(map[string]string)
	}

	deploy.Spec.Template.Annotations["config/checksum"] = s.ConfigMapHash

	if s.Spec.Replicas != nil {
		deploy.Spec.Replicas = s.Spec.Replicas
	}

	deploy.Spec.Selector = &metav1.LabelSelector{MatchLabels: s.AppLabels.AsSelector()}
	deploy.Spec.Template.Spec.ServiceAccountName = serviceAccount
	deploy.Spec.Template.Spec.AutomountServiceAccountToken = new(true)
	deploy.Spec.Template.Spec.Affinity = s.Spec.Affinity
	deploy.Spec.Template.Spec.NodeSelector = s.Spec.NodeSelector
	deploy.Spec.Template.Spec.PriorityClassName = s.Spec.PriorityClassName
	deploy.Spec.Template.Spec.Tolerations = s.Spec.Tolerations
	deploy.Spec.Template.Spec.TopologySpreadConstraints = s.Spec.TopologySpreadConstraints
	deploy.Spec.Template.Spec.Volumes = buildVolumes(s.Spec)
	deploy.Spec.Template.Spec.Containers = []corev1.Container{
		buildContainer(s.Spec, s.Owner.Namespace, getContainer(deploy)),
	}
}

// Build the container for the target allocator. The created container should only cause an update if a mandated value changed.
func buildContainer(spec *dtprometheus.TargetAllocator, namespace string, current corev1.Container) corev1.Container {
	currentLivenessProbe := ptr.Deref(current.LivenessProbe, corev1.Probe{})
	currentReadinessProbe := ptr.Deref(current.ReadinessProbe, corev1.Probe{})

	imagePullPolicy := spec.ImagePullPolicy
	if imagePullPolicy == "" {
		imagePullPolicy = current.ImagePullPolicy
	}

	return corev1.Container{
		Name:            "targetallocator",
		Image:           spec.Image, // TODO: allow using image from fleetmanagement API
		ImagePullPolicy: imagePullPolicy,
		Args:            spec.SanitizedArgs(),
		Ports: []corev1.ContainerPort{
			{Name: insecurePortName, ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
			{Name: securePortName, ContainerPort: 8443, Protocol: corev1.ProtocolTCP},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: configVolume, MountPath: "/conf", ReadOnly: true},
			// TODO: TLS volume
		},
		Env:       k8senv.AppendGoMemoryLimit([]corev1.EnvVar{{Name: "OTELCOL_NAMESPACE", Value: namespace}}, spec.Resources),
		Resources: spec.Resources,
		SecurityContext: &corev1.SecurityContext{
			Privileged:               new(false),
			AllowPrivilegeEscalation: new(false),
			RunAsNonRoot:             new(true),
			RunAsUser:                new(int64(65532)),
			RunAsGroup:               new(int64(65532)),
			ReadOnlyRootFilesystem:   new(true),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"ALL",
				},
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Scheme: corev1.URISchemeHTTP,
					Path:   "/livez",
					Port:   intstr.FromString(insecurePortName),
				},
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
				HTTPGet: &corev1.HTTPGetAction{
					Scheme: corev1.URISchemeHTTP,
					Path:   "/readyz",
					Port:   intstr.FromString(insecurePortName),
				},
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

func buildVolumes(spec *dtprometheus.TargetAllocator) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: configVolume,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: spec.GetDeploymentName(),
					},
					Items: []corev1.KeyToPath{
						{Key: configFile, Path: configFile},
					},
					DefaultMode: new(int32(0o644)),
				},
			},
		},
	}

	// TODO: TLS volume

	return volumes
}

func getContainer(deploy *appsv1.Deployment) corev1.Container {
	if len(deploy.Spec.Template.Spec.Containers) > 0 {
		return deploy.Spec.Template.Spec.Containers[0]
	}

	return corev1.Container{}
}

func safeUnwrap(err error) error {
	if u := errors.Unwrap(err); u != nil {
		return u
	}

	return err
}
