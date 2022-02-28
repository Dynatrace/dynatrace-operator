package kubeobjects

import "sigs.k8s.io/controller-runtime/pkg/client"

func Key(object client.Object) client.ObjectKey {
	return client.ObjectKey{
		Name: object.GetName(), Namespace: object.GetNamespace(),
	}
}
