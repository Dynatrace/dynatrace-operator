package builder

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Option[T client.Object] func(T)

func Build[T client.Object](owner metav1.Object, target T, options ...Option[T]) (T, error) {
	for _, opt := range options {
		opt(target)
	}

	if owner != nil {
		err := controllerutil.SetControllerReference(owner, target, scheme.Scheme)
		if err != nil {
			return target, err
		}
	}

	return target, nil
}

func SetName[T client.Object](name string) func(T) {
	return func(o T) {
		o.SetName(name)
		o.SetResourceVersion("")
	}
}

func SetNamespace[T client.Object](nsName string) func(T) {
	return func(o T) {
		o.SetNamespace(nsName)
		o.SetResourceVersion("")
	}
}

func SetLabels[T client.Object](labels map[string]string) func(T) {
	return func(o T) {
		o.SetLabels(labels)
		o.SetResourceVersion("")
	}
}
