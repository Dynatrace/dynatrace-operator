package dao

import (
	"context"

	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FindServiceAccount(kubeClient client.Client) (*corev1.ServiceAccount, error) {
	serviceAccount := &corev1.ServiceAccount{}
	err := kubeClient.Get(context.TODO(), types.NamespacedName{
		Namespace: _const.DynatraceNamespace,
		Name:      _const.ServiceAccountName,
	}, serviceAccount)

	return serviceAccount, err
}
