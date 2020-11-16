package dtpullsecret

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetImagePullSecret(clt client.Client, instance *v1alpha1.DynaKube) (*corev1.Secret, error) {
	imagePullSecret := &corev1.Secret{}
	err := clt.Get(context.TODO(), client.ObjectKey{Namespace: instance.Namespace, Name: extendWithPullSecretSuffix(instance.Name)}, imagePullSecret)
	if err != nil {
		return nil, err
	}

	return imagePullSecret, err
}
