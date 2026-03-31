package system

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Namespace = "kube-system"
)

func GetUID(ctx context.Context, clt client.Reader) (types.UID, error) {
	kubeSystemNamespace := &corev1.Namespace{}

	err := clt.Get(ctx, client.ObjectKey{Name: Namespace}, kubeSystemNamespace)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return kubeSystemNamespace.UID, nil
}

func IsDeployedViaOLM() bool {
	return os.Getenv(k8senv.OLMOperatorNamespaceEnv) != ""
}
