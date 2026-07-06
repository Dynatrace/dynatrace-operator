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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ContainerName             = "kubemon"
	AnnotationTenantTokenHash = api.InternalFlagPrefix + "kubemon-tenant-token-hash"
	requiredEnvsCapacity      = 4
	storageVolumeName         = "kubemon-storage"
)

var ErrImageRequired = errors.New("kubernetes monitoring image is required")

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
	if !dk.KubernetesMonitoring().IsEnabled() {
		return r.delete(ctx, dk)
	}

	desiredStatefulSet, err := r.buildDesiredStatefulSet(ctx, dk)
	if err != nil {
		return err
	}

	if _, err = r.sts.WithOwner(dk).CreateOrUpdate(ctx, desiredStatefulSet); err != nil {
		return err
	}

	currentStatefulSet, err := r.sts.Get(ctx, types.NamespacedName{Name: desiredStatefulSet.Name, Namespace: desiredStatefulSet.Namespace})
	if k8serrors.IsNotFound(err) || (err == nil && !k8sstatefulset.IsRolloutComplete(currentStatefulSet)) {
		return k8sstatefulset.ErrRolloutInProgress
	}

	return err
}

// buildEnvs prepends the mandatory AG runtime env vars (and optional DT_GROUP) to any user-supplied vars.
func buildEnvs(dk *dynakube.DynaKube) []corev1.EnvVar {
	connInfoCM := dk.KubernetesMonitoring().GetConnectionInfoConfigMapName()

	required := make([]corev1.EnvVar, 0, requiredEnvsCapacity+len(dk.KubernetesMonitoring().Env))
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

	if dk.Spec.KubernetesMonitoring.Group != "" {
		required = append(required, corev1.EnvVar{Name: agconsts.EnvDTGroup, Value: dk.Spec.KubernetesMonitoring.Group})
	}

	return append(required, dk.KubernetesMonitoring().Env...)
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
			},
		},
	}}

	if km.UseEphemeralVolume {
		volumes = append(volumes, corev1.Volume{
			Name:         storageVolumeName,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		})
	}

	return volumes
}

// buildVolumeMounts returns the container-level volume mounts.
func buildVolumeMounts(_ *dynakube.DynaKube) []corev1.VolumeMount {
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
		Name:            ContainerName,
		Image:           image,
		ImagePullPolicy: dk.KubernetesMonitoring().GetPullPolicy(),
		Resources:       dk.KubernetesMonitoring().Resources,
		Env:             buildEnvs(dk),
		VolumeMounts:    buildVolumeMounts(dk),
	}

	km := dk.KubernetesMonitoring()
	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.ActiveGateComponentLabel)

	opts := []k8sstatefulset.Option{
		k8sstatefulset.SetReplicas(replicas),
		k8sstatefulset.SetAllLabels(coreLabels.BuildLabels(), coreLabels.BuildMatchLabels(), coreLabels.BuildLabels(), km.Labels),
		k8sstatefulset.SetAllAnnotations(nil, maputil.MergeMap(km.Annotations, map[string]string{
			AnnotationTenantTokenHash: tokenHash,
		})),
		k8sstatefulset.SetServiceAccount(km.GetServiceAccountName()),
		k8sstatefulset.SetNodeSelector(km.NodeSelector),
		k8sstatefulset.SetTolerations(km.Tolerations),
		k8sstatefulset.SetTopologySpreadConstraints(km.TopologySpreadConstraints),
		k8sstatefulset.SetVolumes(buildVolumes(dk)),
	}

	if km.RollingUpdate != nil {
		opts = append(opts, k8sstatefulset.SetRollingUpdateStrategy(km.RollingUpdate))
	}

	if km.DNSPolicy != "" {
		opts = append(opts, k8sstatefulset.SetDNSPolicy(km.DNSPolicy))
	}

	if km.PriorityClassName != "" {
		opts = append(opts, k8sstatefulset.SetPriorityClassName(km.PriorityClassName))
	}

	if km.TerminationGracePeriodSeconds != nil {
		opts = append(opts, k8sstatefulset.SetTerminationGracePeriodSeconds(km.TerminationGracePeriodSeconds))
	}

	if km.VolumeClaimTemplate != nil {
		opts = append(opts, k8sstatefulset.SetVolumeClaimTemplate(storageVolumeName, *km.VolumeClaimTemplate))
	}

	return k8sstatefulset.Build(dk, km.GetStatefulSetName(), container, opts...)
}

func (r *Reconciler) getTenantTokenHash(ctx context.Context, dk *dynakube.DynaKube) (string, error) {
	var secret corev1.Secret
	if err := r.kubeClient.Get(ctx, types.NamespacedName{Name: dk.KubernetesMonitoring().GetTenantSecretName(), Namespace: dk.Namespace}, &secret); err != nil {
		return "", errors.WithStack(err)
	}

	hash, err := hasher.GenerateHash(string(secret.Data[connectioninfo.TenantTokenKey]))
	if err != nil {
		return "", errors.Wrap(err, "failed to hash tenant token")
	}

	return hash, nil
}
