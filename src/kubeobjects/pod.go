package kubeobjects

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetPod(ctx context.Context, clt client.Reader, name, namespace string) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := clt.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, pod)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return pod, nil
}
