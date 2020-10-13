package dao

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FindKubeSystemUID(kubeClient client.Client) (types.UID, error) {
	kubeSystemNamespace := &corev1.Namespace{}
	err := kubeClient.Get(context.TODO(), client.ObjectKey{Name: "kube-system"}, kubeSystemNamespace)
	if err != nil {
		return "", err
	}
	return kubeSystemNamespace.UID, nil
}
