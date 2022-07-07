package kubesystem

import (
	"context"
	"os"

	cmdConfig "github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const Namespace = "kube-system"

const (
	EnvPodNamespace       = "POD_NAMESPACE"
	EnvPodName            = "POD_NAME"
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

func DeployedViaOLM(clt client.Reader) (bool, error) {
	podName := os.Getenv(EnvPodName)
	podNamespace := os.Getenv(EnvPodNamespace)

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
	kubeCfg, err := cmdConfig.NewKubeConfigProvider().GetConfig()
	if err != nil {
		return nil, err
	}
	return client.New(kubeCfg, client.Options{})
}
