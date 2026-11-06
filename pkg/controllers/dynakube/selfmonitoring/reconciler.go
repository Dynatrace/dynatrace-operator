// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package selfmonitoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/types"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/condition"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sdeployment"
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
	// configKey is the ConfigMap data key holding the rendered OTel Collector config,
	// mounted as self-monitoring.yaml (see --config=/conf/self-monitoring.yaml on the container).
	configKey            = "self-monitoring"
	configFile           = "self-monitoring.yaml"
	configMountDir       = "/conf"
	configVolumeName     = "opentelemetry-collector-configmap"
	configHashAnnotation = "internal.operator.dynatrace.com/self-monitoring-config-hash"

	containerName = "self-monitoring"
	componentName = "self-monitoring"

	trustedCAVolumeMountPath = "/tls/custom/cacerts"
	trustedCAFile            = "rootca.pem"

	otlpHTTPPortName = "otlp-http"
	otlpHTTPPort     = 4318

	healthCheckPortName = "health"
	healthCheckPort     = 13133

	// serviceAccountName and its RBAC are shipped with the Helm chart / manifests.
	serviceAccountName = "dynatrace-self-monitoring"

	// otelCollectorNonRootUser is the "nonroot" UID/GID the collector runs as.
	otelCollectorNonRootUser = 65532

	cacertsVolumeName = "cacerts"

	// Mounted as a directory, not subPath: subPath mounts don't receive live Secret
	// updates, which would break token rotation.
	tokenVolumeName = "dt-token"
	tokenMountPath  = consts.DTComponentsSecretsRootDir + "/tokens"
	tokenFileName   = "data-ingest-token"

	dataIngestTokenMissingMessage = "data ingest token is missing, but it's required for self-monitoring"
	selfMonitoringOffMessage      = "self-monitoring is not enabled, supportability is reduced. Add .spec.selfMonitoring to the DynaKube to enable it."
)

var errDataIngestTokenMissing = errors.New(dataIngestTokenMissingMessage)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
	}
}

type reconcileScope struct {
	// Required for reconcile
	DynaKube    *dynakube.DynaKube
	Spec        *dynakube.SelfMonitoringSpec
	AppLabels   *k8slabel.Labels
	ImageClient image.Client
	// Computed during reconcile
	ConfigMapHash string
	Deployment    *appsv1.Deployment
}

// Reconcile deploys the self-monitoring collector when .spec.selfMonitoring is set and removes it otherwise.
//
// The collector receives OTLP/HTTP from operator-managed components in the DynaKube namespace
// and forwards it to the tenant.
func (r *Reconciler) Reconcile(ctx context.Context, imageClient image.Client, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "self-monitoring")

	if !dk.IsSelfMonitoringEnabled() {
		return r.cleanup(ctx, dk)
	}

	scope := &reconcileScope{
		DynaKube:    dk,
		Spec:        dk.Spec.SelfMonitoring,
		AppLabels:   newAppLabels(dk),
		ImageClient: imageClient,
	}

	err := r.checkDataIngestToken(ctx, dk)
	if errors.Is(err, errDataIngestTokenMissing) {
		k8sconditions.SetDataIngestTokenMissing(dk.Conditions(), dynakube.TokenConditionType, dataIngestTokenMissingMessage)
		condition.Set(dk.Conditions(), dynakube.SelfMonitoringAvailable, componentName, nil, err)
		log.Info("skipping self-monitoring collector deployment", "reason", dataIngestTokenMissingMessage)

		// Not a reconcile error: the user has to provide the token, retrying won't help.
		return nil
	} else if err != nil {
		condition.Set(dk.Conditions(), dynakube.SelfMonitoringAvailable, componentName, nil, err)

		return err
	}

	for _, f := range []func(context.Context, *reconcileScope) error{
		r.reconcileConfigMap,
		r.reconcileDeployment,
		r.reconcileService,
	} {
		if err = f(ctx, scope); err != nil {
			break
		}
	}

	condition.Set(dk.Conditions(), dynakube.SelfMonitoringAvailable, componentName,
		func() bool { return k8sdeployment.IsRolloutComplete(scope.Deployment) }, err)

	return err
}

func newAppLabels(dk *dynakube.DynaKube) *k8slabel.Labels {
	return k8slabel.New(dk.SelfMonitoringName(), dk.Name, "")
}

func (r *Reconciler) checkDataIngestToken(ctx context.Context, dk *dynakube.DynaKube) error {
	tokens, err := token.NewReader(r.apiReader, dk).ReadTokens(ctx)
	if err != nil {
		return fmt.Errorf("read tokens: %w", err)
	}

	if !token.CheckForDataIngestToken(tokens) {
		return errDataIngestTokenMissing
	}

	return nil
}

type imageResolver struct{ *dynakube.DynaKube }

func (r imageResolver) GetImage() string         { return r.Spec.SelfMonitoring.Image }
func (r imageResolver) SetImage(imageURI string) { r.Status.SelfMonitoring.Image = imageURI }

func (r *Reconciler) reconcileConfigMap(ctx context.Context, s *reconcileScope) error {
	log := logd.FromContext(ctx)
	log.Debug("reconciling configmap")

	rendered, err := renderConfig(buildConfigData(s.DynaKube))
	if err != nil {
		return fmt.Errorf("render self-monitoring config: %w", err)
	}

	cm := &corev1.ConfigMap{Name: s.DynaKube.SelfMonitoringName(), Namespace: s.DynaKube.Namespace}

	err = k8sobject.RetryCreateOrUpdate(ctx, r.client, cm, func() error {
		s.AppLabels.MergeInto(cm)
		cm.Data = map[string]string{configKey: rendered}

		return controllerutil.SetControllerReference(s.DynaKube, cm, r.client.Scheme())
	})
	if err != nil {
		return err
	}

	checksum := sha256.Sum256([]byte(rendered))
	s.ConfigMapHash = hex.EncodeToString(checksum[:])

	return nil
}

func buildConfigData(dk *dynakube.DynaKube) selfMonitoringConfigData {
	data := selfMonitoringConfigData{
		Endpoint:           dk.APIURL() + "/v2/otlp",
		Namespace:          dk.Namespace,
		ResourceAttributes: dk.GetResourceAttributes(),
	}

	if dk.Spec.TrustedCAs != "" {
		data.CustomCAPath = filepath.Join(trustedCAVolumeMountPath, trustedCAFile)
	}

	return data
}

func renderConfig(data selfMonitoringConfigData) (string, error) {
	b, err := buildSelfMonitoringOTelConfig(data).Marshal()

	return string(b), err
}

func (r *Reconciler) reconcileDeployment(ctx context.Context, s *reconcileScope) error {
	log := logd.FromContext(ctx)
	log.Debug("reconciling deployment")

	if err := registry.ResolveImage(ctx, imageResolver{s.DynaKube}, s.ImageClient, image.OTelCollector); err != nil {
		return fmt.Errorf("resolve image: %w", err)
	}

	deploy := &appsv1.Deployment{Name: s.DynaKube.SelfMonitoringName(), Namespace: s.DynaKube.Namespace}

	err := k8sobject.RetryCreateOrUpdate(ctx, r.client, deploy, func() error {
		mutateDeployment(deploy, s)

		return controllerutil.SetControllerReference(s.DynaKube, deploy, r.client.Scheme())
	})
	if err != nil {
		return err
	}

	s.Deployment = deploy

	return nil
}

func mutateDeployment(deploy *appsv1.Deployment, s *reconcileScope) {
	s.AppLabels.MergeInto(deploy)

	if deploy.Spec.Template.Labels == nil {
		deploy.Spec.Template.Labels = make(map[string]string)
	}

	// merge instead of replace to keep metadata applied by users or kubectl (e.g. restartedAt)
	maps.Copy(deploy.Spec.Template.Labels, s.Spec.Labels)
	maps.Copy(deploy.Spec.Template.Labels, s.AppLabels.AsSelector())

	if deploy.Spec.Template.Annotations == nil {
		deploy.Spec.Template.Annotations = make(map[string]string)
	}

	// merge instead of replace to keep metadata applied by users or kubectl (e.g. restartedAt)
	maps.Copy(deploy.Spec.Template.Annotations, s.Spec.Annotations)
	deploy.Spec.Template.Annotations[configHashAnnotation] = s.ConfigMapHash

	if s.Spec.Replicas != nil {
		deploy.Spec.Replicas = s.Spec.Replicas
	}

	deploy.Spec.Strategy = k8sdeployment.MergeStrategy(deploy.Spec.Strategy, s.Spec.Strategy)

	deploy.Spec.Selector = &metav1.LabelSelector{MatchLabels: s.AppLabels.AsSelector()}
	deploy.Spec.Template.Spec.ServiceAccountName = serviceAccountName
	deploy.Spec.Template.Spec.AutomountServiceAccountToken = new(true)
	// FSGroup makes the root-owned, 0440 projected token file readable by the non-root collector.
	deploy.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{FSGroup: new(int64(otelCollectorNonRootUser))}

	deploy.Spec.Template.Spec.Affinity = nil
	if s.Spec.Affinity != (corev1.Affinity{}) {
		deploy.Spec.Template.Spec.Affinity = &s.Spec.Affinity
	}

	deploy.Spec.Template.Spec.NodeSelector = s.Spec.NodeSelector
	deploy.Spec.Template.Spec.PriorityClassName = s.Spec.PriorityClassName
	deploy.Spec.Template.Spec.Tolerations = s.Spec.Tolerations
	deploy.Spec.Template.Spec.TopologySpreadConstraints = s.Spec.TopologySpreadConstraints
	deploy.Spec.Template.Spec.ImagePullSecrets = s.DynaKube.CustomPullSecretReferences()
	deploy.Spec.Template.Spec.Volumes = buildVolumes(s)
	// The stored container is passed in so buildContainer can preserve apiserver-defaulted
	// fields (e.g. ImagePullPolicy, probe timeouts) and avoid spurious diffs.
	deploy.Spec.Template.Spec.Containers = []corev1.Container{
		buildContainer(s, k8scontainer.GetFirstInPodSpec(&deploy.Spec.Template.Spec)),
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
		Name:            containerName,
		Image:           s.DynaKube.Status.SelfMonitoring.Image,
		ImagePullPolicy: imagePullPolicy,
		Command:         []string{"/dynatrace-otel-collector"},
		Args:            buildArgs(s),
		Ports: []corev1.ContainerPort{
			{Name: otlpHTTPPortName, ContainerPort: otlpHTTPPort, Protocol: corev1.ProtocolTCP},
			{Name: healthCheckPortName, ContainerPort: healthCheckPort, Protocol: corev1.ProtocolTCP},
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
			HTTPGet:                       &corev1.HTTPGetAction{Scheme: corev1.URISchemeHTTP, Path: "/", Port: intstr.FromInt32(healthCheckPort)},
			InitialDelaySeconds:           15,
			PeriodSeconds:                 20,
			TimeoutSeconds:                currentLivenessProbe.TimeoutSeconds,
			SuccessThreshold:              currentLivenessProbe.SuccessThreshold,
			FailureThreshold:              currentLivenessProbe.FailureThreshold,
			TerminationGracePeriodSeconds: currentLivenessProbe.TerminationGracePeriodSeconds,
		},
		ReadinessProbe: &corev1.Probe{
			HTTPGet:                       &corev1.HTTPGetAction{Scheme: corev1.URISchemeHTTP, Path: "/", Port: intstr.FromInt32(healthCheckPort)},
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

// buildArgs puts the operator-managed config flag first and appends any user-supplied
// args from .spec.selfMonitoring.args after it, so users can layer config overrides on top.
func buildArgs(s *reconcileScope) []string {
	userArgs := s.Spec.SanitizedArgs()

	args := make([]string, 0, 1+len(userArgs))
	args = append(args, "--config="+configMountDir+"/"+configFile)

	return append(args, userArgs...)
}

func buildEnv(s *reconcileScope) []corev1.EnvVar {
	dk := s.DynaKube

	// APIVersion must be set explicitly: the API server defaults it to "v1" on
	// storage, so omitting it here would cause a reconcile diff on every iteration.
	envs := k8senv.AppendGoMemoryLimit([]corev1.EnvVar{
		{
			Name: "MY_POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "status.podIP"},
			},
		},
		{Name: "K8S_CLUSTER_NAME", Value: dk.Status.KubernetesClusterName},
		fieldRefEnv("K8S_NODE_NAME", "spec.nodeName"),
		fieldRefEnv("K8S_POD_NAME", "metadata.name"),
		fieldRefEnv("K8S_NAMESPACE_NAME", "metadata.namespace"),
		fieldRefEnv("K8S_POD_UID", "metadata.uid"),
	}, s.Spec.Resources)

	if dk.HasProxy() {
		envs = append(
			envs,
			proxyEnv("HTTPS_PROXY", dk.Spec.Proxy),
			proxyEnv("HTTP_PROXY", dk.Spec.Proxy),
			// The k8s_attributes processor talks to the API server, which must bypass the proxy.
			corev1.EnvVar{Name: "NO_PROXY", Value: strings.Join([]string{"$(KUBERNETES_SERVICE_HOST)", "kubernetes.default"}, ",")},
		)
	}

	return envs
}

// fieldRefEnv builds an env var sourced from the pod's field path.
// APIVersion is set explicitly to avoid a reconcile diff, as the API server defaults it to "v1".
func fieldRefEnv(name, fieldPath string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: fieldPath},
		},
	}
}

func proxyEnv(name string, src *types.ValueSource) corev1.EnvVar {
	if src.ValueFrom != "" {
		return corev1.EnvVar{
			Name: name,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Name: src.ValueFrom,
					Key:  dynakube.ProxyKey,
				},
			},
		}
	}

	return corev1.EnvVar{Name: name, Value: src.Value}
}

func buildVolumes(s *reconcileScope) []corev1.Volume {
	dk := s.DynaKube

	// DefaultMode is set explicitly (matching what the apiserver defaults it to) so a freshly-built
	// Volume compares equal to the stored, already-defaulted one on the next reconcile.
	defaultMode := new(int32(0o644))

	volumes := []corev1.Volume{
		{
			Name: configVolumeName,
			ConfigMap: &corev1.ConfigMapVolumeSource{
				Name:        dk.SelfMonitoringName(),
				Items:       []corev1.KeyToPath{{Key: configKey, Path: configFile}},
				DefaultMode: defaultMode,
			},
		},
		{
			Name: tokenVolumeName,
			Projected: &corev1.ProjectedVolumeSource{
				DefaultMode: new(int32(0o440)),
				Sources: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							Name: dk.Tokens(),
							Items: []corev1.KeyToPath{
								{Key: token.DataIngestKey, Path: tokenFileName},
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
			ConfigMap: &corev1.ConfigMapVolumeSource{
				Name:        dk.Spec.TrustedCAs,
				Items:       []corev1.KeyToPath{{Key: "certs", Path: trustedCAFile}},
				DefaultMode: defaultMode,
			},
		})
	}

	return append(volumes, s.Spec.Volumes.AsCoreV1()...)
}

func buildVolumeMounts(s *reconcileScope) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{Name: configVolumeName, MountPath: configMountDir, ReadOnly: true},
		{Name: tokenVolumeName, MountPath: tokenMountPath, ReadOnly: true},
	}

	if s.DynaKube.Spec.TrustedCAs != "" {
		mounts = append(mounts, corev1.VolumeMount{Name: cacertsVolumeName, MountPath: trustedCAVolumeMountPath, ReadOnly: true})
	}

	return append(mounts, s.Spec.VolumeMounts...)
}

func (r *Reconciler) reconcileService(ctx context.Context, s *reconcileScope) error {
	logd.FromContext(ctx).Debug("reconciling service")

	svc := &corev1.Service{Name: s.DynaKube.SelfMonitoringName(), Namespace: s.DynaKube.Namespace}

	return k8sobject.RetryCreateOrUpdate(ctx, r.client, svc, func() error {
		s.AppLabels.MergeInto(svc)

		svc.Spec.Type = corev1.ServiceTypeClusterIP
		svc.Spec.Selector = s.AppLabels.AsSelector()
		svc.Spec.Ports = []corev1.ServicePort{
			{
				Name:       otlpHTTPPortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       otlpHTTPPort,
				TargetPort: intstr.FromString(otlpHTTPPortName),
			},
		}

		return controllerutil.SetControllerReference(s.DynaKube, svc, r.client.Scheme())
	})
}

// cleanup removes all self-monitoring objects once .spec.selfMonitoring is unset and sets the
// availability condition to Off, which tells the user that supportability is reduced. The condition
// doubles as the marker of whether something may be deployed: no condition means the DynaKube never
// opted in, Off means the cleanup already happened. Both skip the API calls.
func (r *Reconciler) cleanup(ctx context.Context, dk *dynakube.DynaKube) error {
	cond := meta.FindStatusCondition(*dk.Conditions(), dynakube.SelfMonitoringAvailable)
	if cond == nil || cond.Reason == status.ReasonOff {
		setOffCondition(dk)

		return nil
	}

	log := logd.FromContext(ctx)
	log.Info("self-monitoring disabled, removing collector")

	name := dk.SelfMonitoringName()

	objects := []client.Object{
		&appsv1.Deployment{Name: name, Namespace: dk.Namespace},
		&corev1.Service{Name: name, Namespace: dk.Namespace},
		&corev1.ConfigMap{Name: name, Namespace: dk.Namespace},
	}

	var errs []error

	for _, obj := range objects {
		if err := r.client.Delete(ctx, obj); err != nil && !k8serrors.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("delete %T %s: %w", obj, name, err))
		}
	}

	if err := errors.Join(errs...); err != nil {
		// Keep the condition so the next reconcile retries the cleanup.
		return err
	}

	dk.Status.SelfMonitoring = dynakube.SelfMonitoringStatus{}
	setOffCondition(dk)

	return nil
}

func setOffCondition(dk *dynakube.DynaKube) {
	_ = meta.SetStatusCondition(dk.Conditions(), metav1.Condition{
		Type:    dynakube.SelfMonitoringAvailable,
		Status:  metav1.ConditionFalse,
		Reason:  status.ReasonOff,
		Message: selfMonitoringOffMessage,
	})
}
