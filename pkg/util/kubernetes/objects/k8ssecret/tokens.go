package k8ssecret

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Tokens struct {
	APIToken  string
	PaasToken string
}

func ExtractToken(secret *corev1.Secret, key string) (string, error) {
	value, hasKey := secret.Data[key]
	if !hasKey {
		err := errors.Errorf("missing token %s", key)

		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

func GetDataFromSecretName(ctx context.Context, apiReader client.Reader, namespacedName types.NamespacedName, dataKey string, log logd.Logger) (string, error) {
	query := Query(nil, apiReader, log)

	secret, err := query.Get(ctx, namespacedName)
	if err != nil {
		return "", err
	}

	value, err := ExtractToken(secret, dataKey)
	if err != nil {
		return "", err
	}

	return value, nil
}
