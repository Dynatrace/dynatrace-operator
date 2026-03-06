package k8sserviceentry

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/query"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type QueryObject struct {
	query.Generic[*istiov1beta1.ServiceEntry, *istiov1beta1.ServiceEntryList]
}

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) QueryObject {
	return QueryObject{
		query.Generic[*istiov1beta1.ServiceEntry, *istiov1beta1.ServiceEntryList]{
			Target:     &istiov1beta1.ServiceEntry{},
			ListTarget: &istiov1beta1.ServiceEntryList{},
			ToList: func(list *istiov1beta1.ServiceEntryList) []*istiov1beta1.ServiceEntry {
				out := make([]*istiov1beta1.ServiceEntry, len(list.Items))
				copy(out, list.Items)

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

func isEqual(current, desired *istiov1beta1.ServiceEntry) bool {
	return !hasher.IsAnnotationDifferent(current, desired)
}

func mustRecreate(current, desired *istiov1beta1.ServiceEntry) bool {
	return false
}
