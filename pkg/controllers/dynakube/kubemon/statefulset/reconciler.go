package statefulset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	maputil "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	containerName             = "kubemon"
	annotationTenantTokenHash = api.InternalFlagPrefix + "kubemon-tenant-token-hash"
)

var ErrImageRequired = errors.New("kubernetes monitoring image is required")

type Reconciler struct {
	client       client.Client
	statefulsets k8sstatefulset.QueryObject
}

func NewReconciler(clt client.Client) *Reconciler {
	return &Reconciler{
		client:       clt,
		statefulsets: k8sstatefulset.Query(clt, clt),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	if !dk.KubernetesMonitoring().IsEnabled() {
		return r.delete(ctx, dk)
	}

	desiredStatefulSet, err := r.buildDesiredStatefulSet(ctx, dk)
	if err != nil {
		return err
	}

	if _, err = r.statefulsets.WithOwner(dk).CreateOrUpdate(ctx, desiredStatefulSet); err != nil {
		return err
	}

	currentStatefulSet, err := r.statefulsets.Get(ctx, types.NamespacedName{Name: desiredStatefulSet.Name, Namespace: desiredStatefulSet.Namespace})
	if err != nil {
		return err
	}

	if !k8sstatefulset.IsRolloutComplete(currentStatefulSet) {
		return k8sstatefulset.ErrRolloutInProgress
	}

	return nil
}

// buildEnvs prepends the mandatory AG runtime env vars to any user-supplied vars.
func buildEnvs(dk *dynakube.DynaKube) []corev1.EnvVar {
	const requiredEnvCount = 3

	connInfoCM := dk.KubernetesMonitoring().GetConnectionInfoConfigMapName()

	required := make([]corev1.EnvVar, 0, requiredEnvCount+len(dk.KubernetesMonitoring().Env))
	required = append(required,
		corev1.EnvVar{
			Name:  agconsts.EnvDTCapabilities,
			Value: activegate.KubeMonCapability.ArgumentName,
		},
		corev1.EnvVar{
			Name: connectioninfo.EnvDTTenant,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: connInfoCM},
					Key:                  connectioninfo.TenantUUIDKey,
					Optional:             new(false),
				},
			},
		},
		corev1.EnvVar{
			Name: connectioninfo.EnvDTServer,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: connInfoCM},
					Key:                  connectioninfo.CommunicationEndpointsKey,
					Optional:             new(false),
				},
			},
		},
	)

	return append(required, dk.KubernetesMonitoring().Env...)
}

// buildVolumes returns the pod-level volume for the kubemon-owned tenant token secret.
func buildVolumes(dk *dynakube.DynaKube) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: connectioninfo.TenantSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  dk.KubernetesMonitoring().GetTenantSecretName(),
					DefaultMode: new(int32(0o640)),
				},
			},
		},
	}
}

// buildVolumeMounts returns the container-level volume mounts for the tenant token.
func buildVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      connectioninfo.TenantSecretVolumeName,
			ReadOnly:  true,
			MountPath: connectioninfo.TenantTokenMountPoint,
			SubPath:   connectioninfo.TenantTokenKey,
		},
	}
}

func (r *Reconciler) delete(ctx context.Context, dk *dynakube.DynaKube) error {
	statefulSet := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: dk.KubernetesMonitoring().GetStatefulSetName(), Namespace: dk.Namespace}}

	return r.statefulsets.Delete(ctx, statefulSet)
}

func (r *Reconciler) buildDesiredStatefulSet(ctx context.Context, dk *dynakube.DynaKube) (*appsv1.StatefulSet, error) {
	image := dk.KubernetesMonitoring().GetCustomImage()
	if image == "" {
		return nil, ErrImageRequired
	}

	replicas, err := k8sstatefulset.ResolveReplicas(
		ctx,
		r.client,
		types.NamespacedName{Name: dk.KubernetesMonitoring().GetStatefulSetName(), Namespace: dk.Namespace},
		dk.KubernetesMonitoring().Replicas,
	)
	if err != nil {
		return nil, err
	}

	tokenHash, err := r.getTenantTokenHash(ctx, dk)
	if err != nil {
		return nil, err
	}

	container := corev1.Container{
		Name:            containerName,
		Image:           image,
		ImagePullPolicy: dk.KubernetesMonitoring().GetPullPolicy(),
		Resources:       dk.KubernetesMonitoring().Resources,
		Env:             buildEnvs(dk),
		VolumeMounts:    buildVolumeMounts(),
	}

	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.ActiveGateComponentLabel)

	statefulSet, err := k8sstatefulset.Build(
		dk,
		dk.KubernetesMonitoring().GetStatefulSetName(),
		container,
		k8sstatefulset.SetReplicas(replicas),
		k8sstatefulset.SetAllLabels(coreLabels.BuildLabels(), coreLabels.BuildMatchLabels(), coreLabels.BuildLabels(), dk.KubernetesMonitoring().Labels),
		k8sstatefulset.SetAllAnnotations(nil, maputil.MergeMap(dk.KubernetesMonitoring().Annotations, map[string]string{annotationTenantTokenHash: tokenHash})),
		k8sstatefulset.SetServiceAccount(dk.KubernetesMonitoring().GetServiceAccountName()),
		k8sstatefulset.SetNodeSelector(dk.KubernetesMonitoring().NodeSelector),
		k8sstatefulset.SetTolerations(dk.KubernetesMonitoring().Tolerations),
		k8sstatefulset.SetTopologySpreadConstraints(dk.KubernetesMonitoring().TopologySpreadConstraints),
		k8sstatefulset.SetVolumes(buildVolumes(dk)),
	)
	if err != nil {
		return nil, err
	}

	return statefulSet, nil
}

func (r *Reconciler) getTenantTokenHash(ctx context.Context, dk *dynakube.DynaKube) (string, error) {
	var secret corev1.Secret
	if err := r.client.Get(ctx, types.NamespacedName{Name: dk.KubernetesMonitoring().GetTenantSecretName(), Namespace: dk.Namespace}, &secret); err != nil {
		return "", errors.WithStack(err)
	}

	hash, err := hasher.GenerateHash(string(secret.Data[connectioninfo.TenantTokenKey]))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash tenant token")
	}

	return hash, nil
}
