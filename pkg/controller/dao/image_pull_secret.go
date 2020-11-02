package dao

import (
	"context"
	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetImagePullSecret(clt client.Client, pod *corev1.Pod) (*corev1.Secret, error) {
	imagePullSecret := &corev1.Secret{}
	err := clt.Get(context.TODO(), client.ObjectKey{Namespace: pod.Namespace, Name: _const.ImagePullSecret}, imagePullSecret)
	if err != nil {
		return nil, err
	}

	return imagePullSecret, err
}
