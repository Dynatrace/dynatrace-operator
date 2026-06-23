package statefulset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const containerName = "kubemon"

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

	if err := r.checkPrerequisites(ctx, dk); err != nil {
		return err
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
// Reuses the connection-info ConfigMap produced by the activegate connectioninfo reconciler.
func buildEnvs(dk *dynakube.DynaKube) []corev1.EnvVar {
	const requiredEnvCount = 3

	connInfoCM := dk.ActiveGate().GetConnectionInfoConfigMapName()

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

// buildVolumes returns the pod-level volumes needed for the AG tenant token.
// The secret is produced by the activegate connectioninfo reconciler.
func buildVolumes(dk *dynakube.DynaKube) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: connectioninfo.TenantSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  dk.ActiveGate().GetTenantSecretName(),
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

// checkPrerequisites verifies that the AG-owned resources referenced by the kubemon StatefulSet
// exist before creating it. Both resources are produced by the activegate connectioninfo reconciler.
// Returns a transient error if either is missing — kubemon will retry once AG has reconciled.
//
// NOTE: this is a temporary coupling; long-term kubemon will own its own connectioninfo resources
// so it can run without AG gateway enabled.
func (r *Reconciler) checkPrerequisites(ctx context.Context, dk *dynakube.DynaKube) error {
	cmKey := types.NamespacedName{Name: dk.ActiveGate().GetConnectionInfoConfigMapName(), Namespace: dk.Namespace}
	if err := r.client.Get(ctx, cmKey, &corev1.ConfigMap{}); err != nil {
		if k8serrors.IsNotFound(err) {
			return errors.New("AG connection-info ConfigMap not found; waiting for ActiveGate connectioninfo reconciler")
		}

		return errors.WithStack(err)
	}

	secretKey := types.NamespacedName{Name: dk.ActiveGate().GetTenantSecretName(), Namespace: dk.Namespace}
	if err := r.client.Get(ctx, secretKey, &corev1.Secret{}); err != nil {
		if k8serrors.IsNotFound(err) {
			return errors.New("AG tenant secret not found; waiting for ActiveGate connectioninfo reconciler")
		}

		return errors.WithStack(err)
	}

	return nil
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
		k8sstatefulset.SetAllAnnotations(nil, dk.KubernetesMonitoring().Annotations),
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
