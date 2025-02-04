package secret

import (
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/query"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type QueryObject struct {
	query.Generic[*corev1.Secret, *corev1.SecretList]
}

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) QueryObject {
	return QueryObject{
		query.Generic[*corev1.Secret, *corev1.SecretList]{
			Target:     &corev1.Secret{},
			ListTarget: &corev1.SecretList{},
			ToList: func(sl *corev1.SecretList) []*corev1.Secret {
				out := []*corev1.Secret{}
				for _, s := range sl.Items {
					out = append(out, &s)
				}

				return out
			},
			IsEqual:      isEqual,
			MustRecreate: func(_, _ *corev1.Secret) bool { return false },

			KubeClient: kubeClient,
			KubeReader: kubeReader,
			Log:        log,
		},
	}
}

func isEqual(secret *corev1.Secret, other *corev1.Secret) bool {
	return reflect.DeepEqual(secret.Data, other.Data) && reflect.DeepEqual(secret.Labels, other.Labels) && reflect.DeepEqual(secret.OwnerReferences, other.OwnerReferences)
}
