package dao

import (
	"context"

	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FindKubeSystemUID(kubeClient client.Client) (types.UID, error) {
	kubeSystemNamespace := &corev1.Namespace{}
	err := kubeClient.Get(context.TODO(), client.ObjectKey{Name: _const.KubeSystemNamespace}, kubeSystemNamespace)
	if err != nil {
		return "", err
	}
	return kubeSystemNamespace.UID, nil
}
