package k8sdeployment

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/query"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Query(kubeClient client.Client, kubeReader client.Reader) query.Generic[*appsv1.Deployment, *appsv1.DeploymentList] {
	return query.Generic[*appsv1.Deployment, *appsv1.DeploymentList]{
		Target:     &appsv1.Deployment{},
		ListTarget: &appsv1.DeploymentList{},
		ToList: func(list *appsv1.DeploymentList) []*appsv1.Deployment {
			out := make([]*appsv1.Deployment, len(list.Items))
			for i, item := range list.Items {
				out[i] = &item
			}

			return out
		},
		IsEqual:      isEqual,
		MustRecreate: mustRecreate,

		KubeClient: kubeClient,
		KubeReader: kubeReader,
	}
}

func isEqual(current, desired *appsv1.Deployment) bool {
	// the replicas check is a workaround to enforce the replica count set on the CR
	// without it any direct changes on the deployment will be overseen because the hash will remain the same
	return !hasher.IsAnnotationDifferent(current, desired) && ptr.Deref(desired.Spec.Replicas, 1) == ptr.Deref(current.Spec.Replicas, 1)
}

func mustRecreate(current, desired *appsv1.Deployment) bool {
	return k8slabel.NotEqual(current.Spec.Selector.MatchLabels, desired.Spec.Selector.MatchLabels)
}
