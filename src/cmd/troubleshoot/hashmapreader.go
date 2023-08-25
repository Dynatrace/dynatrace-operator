package troubleshoot

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HashMapReader struct {
	Objects map[string]runtime.Object
}

func (h *HashMapReader) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	objName := fmt.Sprintf("%s/%s", key.Namespace, key.Name)
	if runtimeObj, exists := h.Objects[objName]; exists {
		metaAccessor := meta.NewAccessor()
		err := metaAccessor.SetNamespace(obj, key.Namespace)
		if err != nil {
			return err
		}
		err = metaAccessor.SetName(obj, key.Name)
		if err != nil {
			return err
		}
		if err := scheme.Scheme.Convert(runtimeObj, obj, nil); err != nil {
			return err
		}
		return nil
	}
	return client.IgnoreNotFound(fmt.Errorf("object not found in hashmap"))
}

func (h *HashMapReader) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return nil
}
