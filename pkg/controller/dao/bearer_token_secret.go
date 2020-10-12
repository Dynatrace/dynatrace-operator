package dao

import (
	"context"
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FindBearerTokenSecret(kubeClient client.Client, tokenName string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := kubeClient.Get(context.TODO(), types.NamespacedName{
		Namespace: _const.DynatraceNamespace,
		Name:      tokenName,
	}, secret)

	return secret, err
}
