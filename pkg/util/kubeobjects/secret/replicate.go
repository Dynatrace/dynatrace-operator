package secret

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Replicate will only create the secret once, doesn't mean for keeping the secret up to date
func Replicate(ctx context.Context, query QueryObject, sourceKey, targetKey client.ObjectKey) error {
	secret, err := GetSecretFromSource(ctx, query, sourceKey, targetKey)
	if err != nil {
		return err
	}

	return client.IgnoreAlreadyExists(query.Create(ctx, secret))
}

// GetSecretFromSource retrieves a secret from the source namespace and builds a new secret for the target namespace
func GetSecretFromSource(ctx context.Context, query QueryObject, sourceKey, targetKey client.ObjectKey) (*corev1.Secret, error) {
	sourceSecret, err := query.Get(ctx, sourceKey)
	if err != nil {
		return nil, err
	}

	return BuildForNamespace(targetKey.Name, targetKey.Namespace, sourceSecret.Data, SetLabels(sourceSecret.Labels))
}
