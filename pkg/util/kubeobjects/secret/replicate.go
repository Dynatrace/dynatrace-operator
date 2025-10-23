package secret

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Replicate will only create the secret once, doesn't mean for keeping the secret up to date
func Replicate(ctx context.Context, query QueryObject, sourceSecretName, targetSecretName, sourceNs, targetNs string) error { //nolint:revive
	secret, err := GetSecretFromSource(ctx, query, sourceSecretName, targetSecretName, sourceNs, targetNs)
	if err != nil {
		return err
	}

	return client.IgnoreAlreadyExists(query.Create(ctx, secret))
}

// GetSecretFromSource retrieves a secret from the source namespace and builds a new secret for the target namespace
func GetSecretFromSource(ctx context.Context, query QueryObject, sourceSecretName, targetSecretName, sourceNs, targetNs string) (*corev1.Secret, error) { //nolint:revive
	source, err := query.Get(ctx, types.NamespacedName{Name: sourceSecretName, Namespace: sourceNs})
	if err != nil {
		return nil, err
	}

	return BuildForNamespace(targetSecretName, targetNs, source.Data, SetLabels(source.Labels))
}
