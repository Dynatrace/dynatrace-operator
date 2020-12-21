package dtpullsecret

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetImagePullSecret(clt client.Client, instance *dynatracev1alpha1.DynaKube) (*corev1.Secret, error) {
	imagePullSecret := &corev1.Secret{}
	err := clt.Get(context.TODO(), client.ObjectKey{Namespace: instance.Namespace, Name: extendWithPullSecretSuffix(instance.Name)}, imagePullSecret)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return imagePullSecret, err
}
