package kubesystem

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const Namespace = "kube-system"

func GetUID(clt client.Client) (types.UID, error) {
	kubeSystemNamespace := &corev1.Namespace{}
	err := clt.Get(context.TODO(), client.ObjectKey{Name: Namespace}, kubeSystemNamespace)
	if err != nil {
		return "", err
	}
	return kubeSystemNamespace.UID, nil
}
