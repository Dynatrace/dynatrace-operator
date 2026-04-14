package k8svirtualservice

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/query"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type QueryObject struct {
	query.Generic[*istiov1beta1.VirtualService, *istiov1beta1.VirtualServiceList]
}

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) QueryObject {
	return QueryObject{
		query.Generic[*istiov1beta1.VirtualService, *istiov1beta1.VirtualServiceList]{
			Target:     &istiov1beta1.VirtualService{},
			ListTarget: &istiov1beta1.VirtualServiceList{},
			ToList: func(list *istiov1beta1.VirtualServiceList) []*istiov1beta1.VirtualService {
				return list.Items
			},
			IsEqual:      isEqual,
			MustRecreate: func(_, _ *istiov1beta1.VirtualService) bool { return false },

			KubeClient: kubeClient,
			KubeReader: kubeReader,
		},
	}
}

func isEqual(current, desired *istiov1beta1.VirtualService) bool {
	return !hasher.IsAnnotationDifferent(current, desired)
}
