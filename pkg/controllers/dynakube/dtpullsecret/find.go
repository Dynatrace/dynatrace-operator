package dtpullsecret

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetImagePullSecret(clt client.Client, dk *dynakube.DynaKube) (*corev1.Secret, error) {
	imagePullSecret := &corev1.Secret{}

	err := clt.Get(context.TODO(), client.ObjectKey{Namespace: dk.Namespace, Name: extendWithPullSecretSuffix(dk.Name)}, imagePullSecret)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return imagePullSecret, err
}
