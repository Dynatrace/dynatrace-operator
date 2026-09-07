// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8spod

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Get(ctx context.Context, clt client.Reader, name, namespace string) (*corev1.Pod, error) {
	pod := &corev1.Pod{}

	err := clt.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, pod)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return pod, nil
}

// GetName returns the name of the pod.
// During the webhook injection the pod.Name is not always set yet, in which case it returns the pod.GeneraName
func GetName(pod corev1.Pod) string {
	if pod.Name != "" {
		return pod.Name
	}

	return pod.GenerateName
}

func SetAnnotationIfNotExists(pod *corev1.Pod, key, value string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	if _, ok := pod.Annotations[key]; !ok {
		pod.Annotations[key] = value
	}
}
