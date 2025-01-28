package job

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/query"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
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
			ToList: func(sl *batchv1.JobList) []*batchv1.Job {
				out := []*batchv1.Job{}
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
		},
	}
}

func isEqual(current, desired *batchv1.Job) bool {
	return !hasher.IsAnnotationDifferent(current, desired)
}

func mustRecreate(current, desired *batchv1.Job) bool {
	return labels.NotEqual(current.Spec.Selector.MatchLabels, desired.Spec.Selector.MatchLabels)
}
