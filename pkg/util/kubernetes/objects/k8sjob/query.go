package k8sjob

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/query"
	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type QueryObject struct {
	query.Generic[*batchv1.Job, *batchv1.JobList]
}

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) QueryObject {
	return QueryObject{
		query.Generic[*batchv1.Job, *batchv1.JobList]{
			Target:     &batchv1.Job{},
			ListTarget: &batchv1.JobList{},
			ToList: func(list *batchv1.JobList) []*batchv1.Job {
				out := make([]*batchv1.Job, len(list.Items))
				for i, item := range list.Items {
					out[i] = &item
				}

				return out
			},
			IsEqual:      isEqual,
			MustRecreate: mustRecreate,

			KubeClient: kubeClient,
			KubeReader: kubeReader,
			Log:        log,
		},
	}
}

func isEqual(current, desired *batchv1.Job) bool {
	return !hasher.IsAnnotationDifferent(current, desired)
}

func mustRecreate(current, desired *batchv1.Job) bool {
	return k8slabel.NotEqual(current.Spec.Selector.MatchLabels, desired.Spec.Selector.MatchLabels)
}
