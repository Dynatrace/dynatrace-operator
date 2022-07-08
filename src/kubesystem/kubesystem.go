package kubesystem

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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

func IsDeployedViaOLM(clt client.Reader, podName string, podNamespace string) (bool, error) {
	pod := &corev1.Pod{}
	err := clt.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: podNamespace}, pod)
	if err != nil {
		return false, err
	}
	if _, ok := pod.Annotations[olmSpecificAnnotation]; ok {
		return true, nil
	} else {
		return false, nil
	}
}

func CreateDefaultClient() (client.Client, error) {
	kubeCfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return client.New(kubeCfg, client.Options{})
}
