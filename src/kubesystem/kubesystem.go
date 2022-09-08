package kubesystem

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Namespace             = "kube-system"
	olmSpecificAnnotation = "olm.operatorNamespace"
)

func GetUID(clt client.Reader) (types.UID, error) {
	kubeSystemNamespace := &corev1.Namespace{}
	err := clt.Get(context.TODO(), client.ObjectKey{Name: Namespace}, kubeSystemNamespace)
	if err != nil {
		return "", err
	}
	return kubeSystemNamespace.UID, nil
}

func IsDeployedViaOlm(pod corev1.Pod) bool {
	_, isDeployedViaOlm := pod.Annotations[olmSpecificAnnotation]
	return isDeployedViaOlm
}
