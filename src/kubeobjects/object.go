package kubeobjects

import (
	"context"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Key(object client.Object) client.ObjectKey {
	return client.ObjectKey{
		Name: object.GetName(), Namespace: object.GetNamespace(),
	}
}

func EnsureDeleted(ctx context.Context, client client.Client, obj client.Object) error {
	if err := client.Delete(ctx, obj); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}
