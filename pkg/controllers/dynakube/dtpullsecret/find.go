package dtpullsecret

import (
	"context"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetImagePullSecret(clt client.Client, instance *dynatracev1beta2.DynaKube) (*corev1.Secret, error) {
	imagePullSecret := &corev1.Secret{}

	err := clt.Get(context.TODO(), client.ObjectKey{Namespace: instance.Namespace, Name: extendWithPullSecretSuffix(instance.Name)}, imagePullSecret)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return imagePullSecret, err
}
