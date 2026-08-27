// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"net"
	"strconv"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dtprometheus/condition"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sdeployment"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// scraperConfigKey is the ConfigMap data key holding the rendered OTel Collector config,
	// mounted as scraper.yaml (see --config=/conf/scraper.yaml on the container).
	scraperConfigKey     = "scraper"
	scraperConfigFile    = "scraper.yaml"
	configHashAnnotation = "internal.operator.dynatrace.com/scraper-config-hash"

	trustedCAVolumeMountPath = "/tls/custom/cacerts"
	trustedCAFile            = "rootca.pem"

	healthCheckPortName = "health"
	healthCheckPort     = 13133

	// targetAllocatorPort is the target allocator's plain HTTP port, matching the
	// listen_addr its reconciler writes into its own config.
	targetAllocatorPort = 8080

	// gatewayOTLPPort is the gateway service's OTLP/gRPC port.
	gatewayOTLPPort = 4317

	serviceAccountName = "dynatrace-prometheus-scraper"

	configVolumeName  = "opentelemetry-collector-configmap"
	configMountDir    = "/conf"
	cacertsVolumeName = "cacerts"
)

type Reconciler struct {
	client.Client
}

type reconcileScope struct {
	// Required for reconcile
	Owner       *dtprometheus.DTPrometheus
	DynaKube    *dynakube.DynaKube
	Spec        *dtprometheus.Scraper
	AppLabels   *k8slabel.Labels
	ImageClient image.Client
	// Computed during reconcile
	resolvedImage string
	ConfigMapHash string
	Deployment    *appsv1.Deployment
}

// Reconcile brings the scraper pool in line with the DTPrometheus spec.
//
// The scraper gets no Service: nothing connects to it. The target allocator finds
// its scrapers by listing pods matching AppLabels, and every other connection the
// scraper makes is outbound.
func (r *Reconciler) Reconcile(ctx context.Context, dtp *dtprometheus.DTPrometheus, dk *dynakube.DynaKube, imageClient image.Client) error {
	ctx, _ = logd.NewFromContext(ctx, "scraper")

	scope := &reconcileScope{
		Owner:       dtp,
		DynaKube:    dk,
		Spec:        dtp.Scraper(),
		AppLabels:   k8slabel.OTelScraper(),
		ImageClient: imageClient,
	}

	var err error

	for _, f := range []func(context.Context, *reconcileScope) error{
		r.reconcileConfigMap,
		r.reconcileDeployment,
	} {
		if err = f(ctx, scope); err != nil {
			break
		}
	}

	condition.Set(&scope.Owner.Status.Conditions, dtprometheus.ScraperAvailable, "scraper",
		func() bool { return k8sdeployment.IsRolloutComplete(scope.Deployment) }, err)

	return err
}

// resolveImage uses the explicit image from .spec.scraper.image when set, otherwise
// resolves the latest scraper image from the fleet management API.
func (r *Reconciler) resolveImage(ctx context.Context, s *reconcileScope) error {
	imageURI := s.Spec.Image

	if imageURI == "" {
		var err error

		imageURI, err = registry.ResolveImage(ctx, s.ImageClient, s.Owner.Spec.PublicRegistryOverride, image.Scraper)
		if err != nil {
			return err
		}
	}

	s.resolvedImage = imageURI
	s.Owner.Status.Scraper.ResolvedImage = imageURI

	return nil
}

func (r *Reconciler) reconcileConfigMap(ctx context.Context, s *reconcileScope) error {
	log := logd.FromContext(ctx)
	log.Debug("reconciling configmap")

	rendered, err := renderScraperConfig(buildScraperConfigData(s))
	if err != nil {
		return fmt.Errorf("render scraper config: %w", err)
	}

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: s.Spec.GetDeploymentName(), Namespace: s.Owner.Namespace}}

	err = k8sobject.RetryCreateOrUpdate(ctx, r, cm, func() error {
		s.AppLabels.MergeInto(cm)
		cm.Data = map[string]string{scraperConfigKey: rendered}

		return controllerutil.SetControllerReference(s.Owner, cm, r.Scheme())
	})
	if err != nil {
		return err
	}

	checksum := sha256.Sum256([]byte(rendered))
	s.ConfigMapHash = hex.EncodeToString(checksum[:])

	return nil
}

// buildScraperConfigData resolves the owner-derived inputs to the scraper config. The
// target allocator and gateway both expose Services named after their workload, so
// their endpoints follow from the owning DTPrometheus name.
func buildScraperConfigData(s *reconcileScope) scraperConfigData {
	namespace := s.Owner.Namespace

	return scraperConfigData{
		TargetAllocatorEndpoint: "http://" + net.JoinHostPort(
			serviceFQDN(s.Owner.TargetAllocator().GetDeploymentName(), namespace), strconv.Itoa(targetAllocatorPort)),
		GatewayEndpoint: net.JoinHostPort(
			serviceFQDN(s.Owner.Gateway().GetStatefulSetName(), namespace), strconv.Itoa(gatewayOTLPPort)),
		TargetsPollInterval: s.Spec.TargetsPollInterval.Duration.String(),
	}
}

func serviceFQDN(name, namespace string) string {
	return name + "." + namespace + ".svc.cluster.local"
}

func renderScraperConfig(data scraperConfigData) (string, error) {
	b, err := buildScraperOTelConfig(data).Marshal()

	return string(b), err
}

func (r *Reconciler) reconcileDeployment(ctx context.Context, s *reconcileScope) error {
	log := logd.FromContext(ctx)
	log.Debug("reconciling deployment")

	if err := r.resolveImage(ctx, s); err != nil {
		return fmt.Errorf("resolve image: %w", err)
	}

	deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: s.Spec.GetDeploymentName(), Namespace: s.Owner.Namespace}}

	err := k8sobject.RetryCreateOrUpdate(ctx, r, deploy, func() error {
		mutateDeployment(deploy, s)

		return controllerutil.SetControllerReference(s.Owner, deploy, r.Scheme())
	})
	if err != nil {
		return err
	}

	s.Deployment = deploy

	return nil
}

func mutateDeployment(deploy *appsv1.Deployment, s *reconcileScope) {
	s.AppLabels.MergeInto(deploy)

	deploy.Spec.Template.Labels = maps.Clone(s.Spec.Labels)
	if deploy.Spec.Template.Labels == nil {
		deploy.Spec.Template.Labels = make(map[string]string)
	}

	maps.Copy(deploy.Spec.Template.Labels, s.AppLabels.AsMap())

	deploy.Spec.Template.Annotations = s.Spec.Annotations
	if deploy.Spec.Template.Annotations == nil {
		deploy.Spec.Template.Annotations = make(map[string]string)
	}

	deploy.Spec.Template.Annotations[configHashAnnotation] = s.ConfigMapHash

	if s.Spec.Replicas != nil {
		deploy.Spec.Replicas = s.Spec.Replicas
	}

	if s.Spec.UpdateStrategy.Type != "" {
		deploy.Spec.Strategy = s.Spec.UpdateStrategy
	}

	deploy.Spec.Selector = &metav1.LabelSelector{MatchLabels: s.AppLabels.AsSelector()}
	deploy.Spec.Template.Spec.ServiceAccountName = serviceAccountName
	deploy.Spec.Template.Spec.AutomountServiceAccountToken = new(true)
	deploy.Spec.Template.Spec.Affinity = s.Spec.Affinity
	deploy.Spec.Template.Spec.NodeSelector = s.Spec.NodeSelector
	deploy.Spec.Template.Spec.PriorityClassName = s.Spec.PriorityClassName
	deploy.Spec.Template.Spec.Tolerations = s.Spec.Tolerations
	deploy.Spec.Template.Spec.TopologySpreadConstraints = s.Spec.TopologySpreadConstraints
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
		Name:            "scraper",
		Image:           s.resolvedImage,
		ImagePullPolicy: imagePullPolicy,
		Command:         []string{"/dynatrace-otel-collector"},
		Args:            buildArgs(s),
		Ports: []corev1.ContainerPort{
			{Name: healthCheckPortName, ContainerPort: healthCheckPort, Protocol: corev1.ProtocolTCP},
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

// buildArgs puts the operator-managed config flag first and appends any user-supplied
// args from .spec.scraper.args after it.
func buildArgs(s *reconcileScope) []string {
	userArgs := s.Spec.SanitizedArgs()

	args := make([]string, 0, 1+len(userArgs))
	args = append(args, "--config="+configMountDir+"/"+scraperConfigFile)

	return append(args, userArgs...)
}

func buildEnv(s *reconcileScope) []corev1.EnvVar {
	dk := s.DynaKube

	envs := k8senv.AppendGoMemoryLimit([]corev1.EnvVar{
		// APIVersion must be set explicitly: the API server defaults it to "v1" on
		// storage, so omitting it here would cause a reconcile diff on every iteration.
		{
			Name: "MY_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.name"},
			},
		},
		{
			Name: "MY_POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "status.podIP"},
			},
		},
	}, s.Spec.Resources)

	if dk.HasProxy() {
		envs = append(
			envs,
			proxyEnv("HTTPS_PROXY", dk.Spec.Proxy),
			proxyEnv("HTTP_PROXY", dk.Spec.Proxy),
			corev1.EnvVar{Name: "NO_PROXY", Value: noProxyValue(s)},
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

// noProxyValue keeps the scraper's cluster-internal traffic out of the proxy: the API
// server, the target allocator it polls and the gateway it exports to.
//
// Scrape targets themselves are not covered. Prometheus service discovery addresses
// them by pod IP, which no hostname entry can match, so a user scraping in-cluster
// targets from behind a proxy has to add the pod CIDR to the DynaKube noProxy feature
// flag; that list is appended below.
func noProxyValue(s *reconcileScope) string {
	namespace := s.Owner.Namespace

	values := []string{
		"$(KUBERNETES_SERVICE_HOST)",
		"kubernetes.default",
		serviceFQDN(s.Owner.TargetAllocator().GetDeploymentName(), namespace),
		serviceFQDN(s.Owner.Gateway().GetStatefulSetName(), namespace),
	}

	for hostname := range strings.SplitSeq(s.DynaKube.FF().GetNoProxy(), ",") {
		if hostname = strings.TrimSpace(hostname); hostname != "" {
			values = append(values, hostname)
		}
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
					LocalObjectReference: corev1.LocalObjectReference{Name: s.Spec.GetDeploymentName()},
					Items:                []corev1.KeyToPath{{Key: scraperConfigKey, Path: scraperConfigFile}},
					DefaultMode:          defaultMode,
				},
			},
		},
	}

	// Mounted so scrape configs can point their tlsConfig at it; the collector config
	// itself does not reference the bundle.
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
	mounts := []corev1.VolumeMount{
		{Name: configVolumeName, MountPath: configMountDir, ReadOnly: true},
	}

	if s.DynaKube.Spec.TrustedCAs != "" {
		mounts = append(mounts, corev1.VolumeMount{Name: cacertsVolumeName, MountPath: trustedCAVolumeMountPath, ReadOnly: true})
	}

	return mounts
}
