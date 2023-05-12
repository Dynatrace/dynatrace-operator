package kubeobjects

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProbeResult int

const (
	ProbeObjectFound ProbeResult = iota
	ProbeObjectNotFound
	ProbeTypeFound
	ProbeTypeNotFound
	ProbeUnknown
)

func KubernetesObjectProbe(gvk schema.GroupVersionKind,
	namespace string, name string, config *rest.Config) error {
	var objQuery unstructured.Unstructured
	objQuery.Object = make(map[string]any)

	objQuery.SetGroupVersionKind(gvk)

	runtimeClient, err := client.New(config, client.Options{})
	if err != nil {
		return err
	}
	if name == "" {
		err = runtimeClient.List(context.TODO(), &objQuery, client.InNamespace(namespace))
	} else {
		err = runtimeClient.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, &objQuery)
	}

	return err
}
