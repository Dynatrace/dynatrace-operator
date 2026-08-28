// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package statefulset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/version"
	operatorconsts "github.com/Dynatrace/dynatrace-operator/pkg/consts"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/statefulset/probe"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	kubemonauthtoken "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/authtoken"
	kubemoncustomproperties "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8ssecuritycontext"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	maputil "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ContainerName                  = "kubemon"
	AnnotationTenantTokenHash      = api.InternalFlagPrefix + "kubemon-tenant-token-hash"
	AnnotationAuthTokenHash        = api.InternalFlagPrefix + "kubemon-authtoken-hash"
	AnnotationCustomPropertiesHash = api.InternalFlagPrefix + "kubemon-customproperties-hash"
	AnnotationKSPMTokenHash        = api.InternalFlagPrefix + "kubemon-kspm-token-hash"
	AnnotationTLSSecretHash        = api.InternalFlagPrefix + "kubemon-tls-secret-hash"
	StorageVolumeName              = "kubemon-storage"
	AuthTokenVolumeName            = "kubemon-authtoken-secret"
	kspmTokenVolumeName            = "kspm-token"
	kspmTokenMountPath             = operatorconsts.DTComponentsSecretsRootDir + "/tokens/kspm/node-configuration-collector"
)

var (
	ErrMissingKubeSystemUID = errors.New("kube-system UUID not yet available")
	ErrMissingTenantToken   = errors.New("tenant token secret is missing or has an empty value")
	ErrMissingAuthToken     = errors.New("auth token secret is missing or has an empty value")
)

type Reconciler struct {
	kubeClient client.Client
	sts        k8sstatefulset.QueryObject
}

func NewReconciler(kubeClient client.Client) *Reconciler {
	return &Reconciler{
		kubeClient: kubeClient,
		sts:        k8sstatefulset.Query(kubeClient, kubeClient),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube, imageClient image.Client, versionClient version.Client) error {
	ctx, _ = logd.NewFromContext(ctx, "statefulset")

	if !dk.KubernetesMonitoring().IsEnabled() {
		return r.delete(ctx, dk)
	}

	if err := ensureReady(dk); err != nil {
		return err
	}

	desiredStatefulSet, err := r.buildDesiredStatefulSet(ctx, dk, imageClient, versionClient)
	if err != nil {
		return err
	}

	_, err = r.sts.CreateOrUpdate(ctx, desiredStatefulSet)
	if err != nil {
		return err
	}

	currentStatefulSet, err := r.sts.Get(ctx, client.ObjectKey{Name: desiredStatefulSet.Name, Namespace: desiredStatefulSet.Namespace})
	if k8serrors.IsNotFound(err) || (err == nil && !k8sstatefulset.IsRolloutComplete(currentStatefulSet)) {
		return k8sstatefulset.ErrRolloutInProgress
	}

	return err
}

// ensureReady validates the DynaKube fields required to build the StatefulSet. Without any of
// them the build fails either way, so we short-circuit here before doing any work or API calls.
func ensureReady(dk *dynakube.DynaKube) error {
	if dk.Status.KubeSystemUUID == "" {
		return ErrMissingKubeSystemUID
	}

	return nil
}

func buildPodAnnotations(dk *dynakube.DynaKube, tokenHash, authTokenHash, customPropertiesHash string) map[string]string {
	annotations := map[string]string{
		AnnotationTenantTokenHash:              tokenHash,
		AnnotationAuthTokenHash:                authTokenHash,
		AnnotationCustomPropertiesHash:         customPropertiesHash,
		mutator.AnnotationInjectionSplitMounts: "true",
	}

	if dk.KSPM().IsEnabled() {
		annotations[AnnotationKSPMTokenHash] = dk.KSPM().TokenSecretHash
		annotations[AnnotationTLSSecretHash] = dk.KubernetesMonitoring().TLSSecretHash
	}

	return annotations
}

// buildEnvs prepends the mandatory AG runtime env vars (and optional DT_GROUP) to any user-supplied vars.
func buildEnvs(dk *dynakube.DynaKube) []corev1.EnvVar {
	connInfoCM := dk.KubernetesMonitoring().GetConnectionInfoConfigMapName()

	envs := []corev1.EnvVar{
		{
			Name:  agconsts.EnvDTCapabilities,
			Value: activegate.KubeMonCapability.ArgumentName,
		},
		{
			Name:  agconsts.EnvDTIDSeedNamespace,
			Value: dk.Namespace,
		},
		{
			Name:  agconsts.EnvDTIDSeedClusterID,
			Value: dk.Status.KubeSystemUUID,
		},
		{
			Name: deploymentmetadata.EnvDTDeploymentMetadata,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: deploymentmetadata.GetDeploymentMetadataConfigMapName(dk.Name),
					},
					Key:      deploymentmetadata.KubemonMetadataKey,
					Optional: new(false),
				},
			},
		},
		{
			Name: connectioninfo.EnvDTTenant,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: connInfoCM},
					Key:                  connectioninfo.TenantUUIDKey,
					Optional:             new(false),
				},
			},
		},
		{
			Name: connectioninfo.EnvDTServer,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: connInfoCM},
					Key:                  connectioninfo.CommunicationEndpointsKey,
					Optional:             new(false),
				},
			},
		},
	}

	if dk.Spec.KubernetesMonitoring.Group != "" {
		envs = append(envs, corev1.EnvVar{Name: agconsts.EnvDTGroup, Value: dk.Spec.KubernetesMonitoring.Group})
	}

	return append(envs, dk.KubernetesMonitoring().Env...)
}

// buildVolumes returns the pod-level volumes.
func buildVolumes(dk *dynakube.DynaKube) []corev1.Volume {
	km := dk.KubernetesMonitoring()
	volumes := []corev1.Volume{
		{
			Name: connectioninfo.TenantSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  km.GetTenantSecretName(),
					DefaultMode: new(int32(0o640)),
					Optional:    new(false),
				},
			},
		},
		{
			Name: AuthTokenVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  km.GetAuthTokenSecretName(),
					DefaultMode: new(int32(0o640)),
					Optional:    new(false),
				},
			},
		},
		{
			Name:         StorageVolumeName,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
		{
			Name: agconsts.GatewayLibTempVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: agconsts.GatewayDataVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: agconsts.GatewayLogVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: agconsts.GatewayConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: agconsts.TrustStoreVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: agconsts.GatewaySslVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name:         agconsts.InitCertLoaderWorkDirVolumeName,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
	}

	if km.CustomProperties != nil {
		volumes = append(volumes, corev1.Volume{
			Name: kubemoncustomproperties.VolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  km.GetCustomPropertiesSecretName(),
					DefaultMode: new(int32(0o640)),
					Optional:    new(false),
				},
			},
		})
	}

	if dk.KSPM().IsEnabled() {
		volumes = append(volumes,
			corev1.Volume{
				Name: kspmTokenVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.KSPM().GetTokenSecretName(),
						DefaultMode: new(int32(0o640)),
					},
				},
			},
			corev1.Volume{
				Name: agconsts.CertsVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.KubernetesMonitoring().GetTLSSecretName(),
						DefaultMode: new(int32(0o640)),
					},
				},
			})
	}

	return volumes
}

// buildVolumeMounts returns the container-level volume mounts.
func buildVolumeMounts(dk *dynakube.DynaKube) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{
			Name:      connectioninfo.TenantSecretVolumeName,
			ReadOnly:  true,
			MountPath: connectioninfo.TenantTokenMountPoint,
			SubPath:   connectioninfo.TenantTokenKey,
		},
		{
			Name:      AuthTokenVolumeName,
			ReadOnly:  true,
			MountPath: agconsts.AuthTokenMountPoint,
			SubPath:   kubemonauthtoken.SecretKey,
		},
		{
			Name:      StorageVolumeName,
			MountPath: agconsts.GatewayTmpMountPath,
		},
		{
			ReadOnly:  false,
			Name:      agconsts.GatewaySslVolumeName,
			MountPath: agconsts.GatewaySslMountPath,
		},
		{
			ReadOnly:  false,
			Name:      agconsts.GatewayLibTempVolumeName,
			MountPath: agconsts.GatewayLibTempMountPath,
		},
		{
			ReadOnly:  false,
			Name:      agconsts.GatewayDataVolumeName,
			MountPath: agconsts.GatewayDataMountPath,
		},
		{
			ReadOnly:  false,
			Name:      agconsts.GatewayLogVolumeName,
			MountPath: agconsts.GatewayLogMountPath,
		},
		{
			ReadOnly:  false,
			Name:      agconsts.GatewayConfigVolumeName,
			MountPath: agconsts.GatewayConfigMountPath,
		},
		{
			ReadOnly:  true,
			Name:      agconsts.TrustStoreVolumeName,
			MountPath: agconsts.TrustStoreCacertsMountPath,
			SubPath:   agconsts.K8sCertificateFile,
		},
	}

	if dk.KubernetesMonitoring().CustomProperties != nil {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      kubemoncustomproperties.VolumeName,
			ReadOnly:  true,
			MountPath: kubemoncustomproperties.MountPath,
			SubPath:   kubemoncustomproperties.DataKey,
		})
	}

	if dk.KSPM().IsEnabled() {
		mounts = append(mounts,
			corev1.VolumeMount{
				Name:      kspmTokenVolumeName,
				ReadOnly:  true,
				MountPath: kspmTokenMountPath,
				SubPath:   kspm.TokenSecretKey,
			},
			corev1.VolumeMount{
				Name:      agconsts.CertsVolumeName,
				ReadOnly:  true,
				MountPath: agconsts.CertsMountPath,
			})
	}

	return mounts
}

func buildInitVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  false,
			Name:      agconsts.TrustStoreVolumeName,
			MountPath: agconsts.GatewaySslMountPath,
		},
		{
			ReadOnly:  false,
			Name:      agconsts.InitCertLoaderWorkDirVolumeName,
			MountPath: agconsts.InitCertLoaderWorkDirMountPath,
		},
	}
}

func (r *Reconciler) delete(ctx context.Context, dk *dynakube.DynaKube) error {
	statefulSet := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: dk.KubernetesMonitoring().GetStatefulSetName(), Namespace: dk.Namespace}}

	return r.sts.Delete(ctx, statefulSet)
}

func (r *Reconciler) resolveImage(ctx context.Context, dk *dynakube.DynaKube, imageClient image.Client, versionClient version.Client) (string, error) {
	var (
		imageURI string
		err      error
	)

	switch {
	case dk.KubernetesMonitoring().GetCustomImage() != "":
		imageURI = dk.KubernetesMonitoring().GetCustomImage()
	case dk.FF().IsPublicRegistry():
		var imageInfo *image.Info

		imageInfo, err = imageClient.GetComponentLatestInfo(ctx, image.ActiveGate, dk.PublicRegistryOverride())
		if err != nil {
			return "", err
		}

		imageURI = imageInfo.URI
	default:
		var latestVersion string

		latestVersion, err = versionClient.GetLatestActiveGateVersion(ctx, installer.OSUnix)
		if err != nil {
			return "", err
		}

		imageURI = dk.KubernetesMonitoring().GetDefaultImage(latestVersion)
	}

	dk.KubernetesMonitoring().ResolvedImage = imageURI

	return imageURI, nil
}

func (r *Reconciler) buildDesiredStatefulSet(ctx context.Context, dk *dynakube.DynaKube, imageClient image.Client, versionClient version.Client) (*appsv1.StatefulSet, error) {
	imageURI, err := r.resolveImage(ctx, dk, imageClient, versionClient)
	if err != nil {
		return nil, err
	}

	// no .replicas means the field is controlled by HPA, so we read it from live object
	replicas, err := k8sstatefulset.ResolveReplicas(
		ctx,
		r.kubeClient,
		client.ObjectKey{Name: dk.KubernetesMonitoring().GetStatefulSetName(), Namespace: dk.Namespace},
		dk.KubernetesMonitoring().Replicas,
	)
	if err != nil {
		return nil, err
	}

	tokenHash, err := r.getTenantTokenHash(ctx, dk)
	if err != nil {
		return nil, err
	}

	authTokenHash, err := r.getAuthTokenHash(ctx, dk)
	if err != nil {
		return nil, err
	}

	customPropertiesHash, err := r.getCustomPropertiesHash(ctx, dk)
	if err != nil {
		return nil, err
	}

	initContainer := corev1.Container{
		Name:            agconsts.InitContainerTemplateName,
		Image:           imageURI,
		ImagePullPolicy: dk.KubernetesMonitoring().ImagePullPolicy,
		WorkingDir:      agconsts.InitCertLoaderWorkDirMountPath,
		Command:         []string{"/bin/bash"},
		Args:            []string{"-c", agconsts.K8scrt2jksPath},
		VolumeMounts:    buildInitVolumeMounts(),
		Resources:       dk.KubernetesMonitoring().Resources,
		SecurityContext: buildSecurityContext(dk.KubernetesMonitoring().Annotations, agconsts.InitContainerTemplateName),
	}

	container := corev1.Container{
		Name:            ContainerName,
		Image:           imageURI,
		ImagePullPolicy: dk.KubernetesMonitoring().ImagePullPolicy,
		Resources:       dk.KubernetesMonitoring().Resources,
		Env:             buildEnvs(dk),
		VolumeMounts:    buildVolumeMounts(dk),
		ReadinessProbe:  probe.Readiness(),
		LivenessProbe:   probe.Liveness(),
		Ports: []corev1.ContainerPort{
			{Name: agconsts.HTTPSServicePortName, ContainerPort: agconsts.HTTPSContainerPort},
			{Name: agconsts.HTTPServicePortName, ContainerPort: agconsts.HTTPContainerPort},
		},
		SecurityContext: buildSecurityContext(dk.KubernetesMonitoring().Annotations, ContainerName),
	}

	km := dk.KubernetesMonitoring()

	km.Annotations = k8ssecuritycontext.RemoveAppArmorAnnotation(km.Annotations, agconsts.InitContainerTemplateName)
	km.Annotations = k8ssecuritycontext.RemoveAppArmorAnnotation(km.Annotations, ContainerName)

	labels := k8slabel.New(k8slabel.KubeMonComponentLabel, dk.GetName(), "")

	opts := []k8sstatefulset.Option{
		k8sstatefulset.SetReplicas(replicas),
		k8sstatefulset.SetAllLabels(labels.AsMap(), labels.AsSelector(), labels.AsMap(), km.Labels),
		k8sstatefulset.SetAllAnnotations(nil, maputil.MergeMap(km.Annotations, buildPodAnnotations(dk, tokenHash, authTokenHash, customPropertiesHash))),
		k8sstatefulset.SetServiceAccount(km.GetServiceAccountName()),
		k8sstatefulset.SetNodeSelector(km.NodeSelector),
		k8sstatefulset.SetTolerations(km.Tolerations),
		k8sstatefulset.SetTopologySpreadConstraints(km.TopologySpreadConstraints),
		k8sstatefulset.SetVolumes(buildVolumes(dk)),
		k8sstatefulset.SetRollingUpdateStrategy(km.RollingUpdate),
		k8sstatefulset.SetDNSPolicy(km.DNSPolicy),
		k8sstatefulset.SetPriorityClassName(km.PriorityClassName),
		k8sstatefulset.SetTerminationGracePeriodSeconds(km.TerminationGracePeriodSeconds),
		k8sstatefulset.SetImagePullSecrets(dk.ImagePullSecretReferences()),
		k8sstatefulset.SetAutomountServiceAccountToken(true),
		k8sstatefulset.SetSecurityContext(buildPodSecurityContext()),
		k8sstatefulset.SetInitContainer(initContainer),
	}

	return k8sstatefulset.Build(dk, km.GetStatefulSetName(), container, opts...)
}

func (r *Reconciler) getTenantTokenHash(ctx context.Context, dk *dynakube.DynaKube) (string, error) {
	var secret corev1.Secret
	if err := r.kubeClient.Get(ctx, client.ObjectKey{Name: dk.KubernetesMonitoring().GetTenantSecretName(), Namespace: dk.Namespace}, &secret); err != nil {
		return "", errors.WithStack(err)
	}

	token := secret.Data[connectioninfo.TenantTokenKey]
	if len(token) == 0 {
		return "", ErrMissingTenantToken
	}

	hash, err := hasher.GenerateHash(string(token))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash tenant token")
	}

	return hash, nil
}

func (r *Reconciler) getAuthTokenHash(ctx context.Context, dk *dynakube.DynaKube) (string, error) {
	var secret corev1.Secret
	if err := r.kubeClient.Get(ctx, client.ObjectKey{Name: dk.KubernetesMonitoring().GetAuthTokenSecretName(), Namespace: dk.Namespace}, &secret); err != nil {
		return "", errors.WithStack(err)
	}

	token := secret.Data[kubemonauthtoken.SecretKey]
	if len(token) == 0 {
		return "", ErrMissingAuthToken
	}

	hash, err := hasher.GenerateHash(string(token))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash auth token")
	}

	return hash, nil
}

// getCustomPropertiesHash returns "" if Secret is missing or empty.
// tenant/auth tokens are required and would throw error, custom properties are optional.
func (r *Reconciler) getCustomPropertiesHash(ctx context.Context, dk *dynakube.DynaKube) (string, error) {
	var secret corev1.Secret

	err := r.kubeClient.Get(ctx, client.ObjectKey{Name: dk.KubernetesMonitoring().GetCustomPropertiesSecretName(), Namespace: dk.Namespace}, &secret)
	if k8serrors.IsNotFound(err) {
		return "", nil
	}

	if err != nil {
		return "", errors.WithStack(err)
	}

	data := secret.Data[kubemoncustomproperties.DataKey]
	if len(data) == 0 {
		return "", nil
	}

	hash, err := hasher.GenerateSecureHash(string(data))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash custom properties")
	}

	return hash, nil
}

func buildSecurityContext(annotations map[string]string, containerName string) *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged:               new(false),
		AllowPrivilegeEscalation: new(false),
		RunAsNonRoot:             new(true),
		RunAsUser:                new(agconsts.DockerImageUser),
		RunAsGroup:               new(agconsts.DockerImageGroup),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		ReadOnlyRootFilesystem: new(true),
		AppArmorProfile:        k8ssecuritycontext.GetAppArmorProfile(annotations, containerName),
	}
}

func buildPodSecurityContext() *corev1.PodSecurityContext {
	sc := corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	sc.FSGroup = new(agconsts.DockerImageGroup)

	return &sc
}
