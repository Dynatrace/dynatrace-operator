// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package statefulset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	kubemonauthtoken "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	maputil "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ContainerName             = "kubemon"
	AnnotationTenantTokenHash = api.InternalFlagPrefix + "kubemon-tenant-token-hash"
	AnnotationAuthTokenHash   = api.InternalFlagPrefix + "kubemon-authtoken-hash"
	StorageVolumeName         = "kubemon-storage"
	AuthTokenVolumeName       = "kubemon-authtoken-secret"
)

var (
	ErrImageRequired        = errors.New("kubernetes monitoring image is required")
	ErrMissingKubeSystemUID = errors.New("kube-system UUID not yet available")
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

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	ctx, _ = logd.NewFromContext(ctx, "kubemon-statefulset")

	if !dk.KubernetesMonitoring().IsEnabled() {
		return r.delete(ctx, dk)
	}

	desiredStatefulSet, err := r.buildDesiredStatefulSet(ctx, dk)
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

// buildEnvs prepends the mandatory AG runtime env vars (and optional DT_GROUP) to any user-supplied vars.
func buildEnvs(dk *dynakube.DynaKube) ([]corev1.EnvVar, error) {
	if dk.Status.KubeSystemUUID == "" {
		return nil, ErrMissingKubeSystemUID
	}

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

	return append(envs, dk.KubernetesMonitoring().Env...), nil
}

// buildVolumes returns the pod-level volumes.
func buildVolumes(dk *dynakube.DynaKube) []corev1.Volume {
	km := dk.KubernetesMonitoring()
	volumes := []corev1.Volume{{
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
	}

	return volumes
}

// buildVolumeMounts returns the container-level volume mounts.
func buildVolumeMounts(_ *dynakube.DynaKube) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{{
		Name:      connectioninfo.TenantSecretVolumeName,
		ReadOnly:  true,
		MountPath: connectioninfo.TenantTokenMountPoint,
		SubPath:   connectioninfo.TenantTokenKey,
	}, {
		Name:      AuthTokenVolumeName,
		ReadOnly:  true,
		MountPath: agconsts.AuthTokenMountPoint,
		SubPath:   kubemonauthtoken.SecretKey,
	}, {
		Name:      StorageVolumeName,
		MountPath: agconsts.GatewayTmpMountPoint,
	}}

	return mounts
}

func (r *Reconciler) delete(ctx context.Context, dk *dynakube.DynaKube) error {
	statefulSet := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: dk.KubernetesMonitoring().GetStatefulSetName(), Namespace: dk.Namespace}}

	return r.sts.Delete(ctx, statefulSet)
}

func (r *Reconciler) buildDesiredStatefulSet(ctx context.Context, dk *dynakube.DynaKube) (*appsv1.StatefulSet, error) {
	image := dk.KubernetesMonitoring().GetCustomImage()
	if image == "" {
		return nil, ErrImageRequired
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

	envs, err := buildEnvs(dk)
	if err != nil {
		return nil, err
	}

	container := corev1.Container{
		Name:            ContainerName,
		Image:           image,
		ImagePullPolicy: dk.KubernetesMonitoring().GetPullPolicy(),
		Resources:       dk.KubernetesMonitoring().Resources,
		Env:             envs,
		VolumeMounts:    buildVolumeMounts(dk),
	}

	km := dk.KubernetesMonitoring()
	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.KubeMonComponentLabel)

	opts := []k8sstatefulset.Option{
		k8sstatefulset.SetReplicas(replicas),
		k8sstatefulset.SetAllLabels(coreLabels.BuildLabels(), coreLabels.BuildMatchLabels(), coreLabels.BuildLabels(), km.Labels),
		k8sstatefulset.SetAllAnnotations(nil, maputil.MergeMap(km.Annotations, map[string]string{
			AnnotationTenantTokenHash: tokenHash,
			AnnotationAuthTokenHash:   authTokenHash,
		})),
		k8sstatefulset.SetServiceAccount(km.GetServiceAccountName()),
		k8sstatefulset.SetNodeSelector(km.NodeSelector),
		k8sstatefulset.SetTolerations(km.Tolerations),
		k8sstatefulset.SetTopologySpreadConstraints(km.TopologySpreadConstraints),
		k8sstatefulset.SetVolumes(buildVolumes(dk)),
		k8sstatefulset.SetRollingUpdateStrategy(km.RollingUpdate),
		k8sstatefulset.SetDNSPolicy(km.DNSPolicy),
		k8sstatefulset.SetPriorityClassName(km.PriorityClassName),
		k8sstatefulset.SetTerminationGracePeriodSeconds(km.TerminationGracePeriodSeconds),
		k8sstatefulset.SetAutomountServiceAccountToken(true),
	}

	return k8sstatefulset.Build(dk, km.GetStatefulSetName(), container, opts...)
}

func (r *Reconciler) getTenantTokenHash(ctx context.Context, dk *dynakube.DynaKube) (string, error) {
	var secret corev1.Secret
	if err := r.kubeClient.Get(ctx, client.ObjectKey{Name: dk.KubernetesMonitoring().GetTenantSecretName(), Namespace: dk.Namespace}, &secret); err != nil {
		return "", errors.WithStack(err)
	}

	hash, err := hasher.GenerateHash(string(secret.Data[connectioninfo.TenantTokenKey]))
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

	hash, err := hasher.GenerateHash(string(secret.Data[kubemonauthtoken.SecretKey]))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash auth token")
	}

	return hash, nil
}
