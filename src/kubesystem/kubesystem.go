package kubesystem

import (
	"context"

	"github.com/pkg/errors"
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

func IsDeployedViaOlm(clt client.Reader, podName string, podNamespace string) (bool, error) {
	pod := &corev1.Pod{}
	err := clt.Get(context.TODO(), types.NamespacedName{Name: podName, Namespace: podNamespace}, pod)
	if err != nil {
		return false, errors.WithStack(err)
	}
	_, isDeployedViaOlm := pod.Annotations[olmSpecificAnnotation]
	return isDeployedViaOlm, nil
}

func CreateDefaultClient() (client.Client, error) {
	kubeCfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	kubeClient, err := client.New(kubeCfg, client.Options{})
	return kubeClient, errors.WithStack(err)
}
