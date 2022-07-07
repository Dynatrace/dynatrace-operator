package kubesystem

import (
	"context"
	"os"

	cmdConfig "github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
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

func DeployedViaOLM() (bool, error) {
	kubeCfg, err := cmdConfig.NewKubeConfigProvider().GetConfig()
	if err != nil {
		return false, err
	}
	clientset, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		return false, err
	}

	podName := os.Getenv(EnvPodName)
	podNamespace := os.Getenv(EnvPodNamespace)

	pod, err := clientset.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if _, ok := pod.Annotations[olmSpecificAnnotation]; ok {
		return true, nil
	} else {
		return false, nil
	}
}
