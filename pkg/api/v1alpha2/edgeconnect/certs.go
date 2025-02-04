package edgeconnect

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const TrustedCAKey = "certs"

func (ec *EdgeConnect) TrustedCAs(ctx context.Context, kubeReader client.Reader) ([]byte, error) {
	configName := ec.Spec.CaCertsRef
	if configName != "" {
		var caConfigMap corev1.ConfigMap

		err := kubeReader.Get(ctx, client.ObjectKey{Name: configName, Namespace: ec.Namespace}, &caConfigMap)
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("failed to get trustedCa from %s configmap", configName))
		}

		return []byte(caConfigMap.Data[TrustedCAKey]), nil
	}

	return nil, nil
}
