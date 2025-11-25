package k8sdeployment

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/query"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) query.Generic[*appsv1.Deployment, *appsv1.DeploymentList] {
	return query.Generic[*appsv1.Deployment, *appsv1.DeploymentList]{
		Target:     &appsv1.Deployment{},
		ListTarget: &appsv1.DeploymentList{},
		ToList: func(sl *appsv1.DeploymentList) []*appsv1.Deployment {
			out := []*appsv1.Deployment{}
			for _, s := range sl.Items {
				out = append(out, &s)
			}

			return out
		},
		IsEqual:      isEqual,
		MustRecreate: mustRecreate,

		KubeClient: kubeClient,
		KubeReader: kubeReader,
		Log:        log,
	}
}

func isEqual(current, desired *appsv1.Deployment) bool {
	return !hasher.IsAnnotationDifferent(current, desired)
}

func mustRecreate(current, desired *appsv1.Deployment) bool {
	return k8slabel.NotEqual(current.Spec.Selector.MatchLabels, desired.Spec.Selector.MatchLabels)
}
