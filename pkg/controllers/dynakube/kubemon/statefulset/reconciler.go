package statefulset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
		Env:             dk.KubernetesMonitoring().Env,
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
	)
	if err != nil {
		return nil, err
	}

	return statefulSet, nil
}
