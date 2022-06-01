package query

import (
	"context"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	trustedCAKey = "certs"
	proxyKey     = "proxy"
	tlsCertKey   = "server.crt"
)

type DynakubeQuery struct {
	clt       client.Client
	namespace string
}

func NewDynakubeQuery(clt client.Client, namespace string) *DynakubeQuery {
	return &DynakubeQuery{
		clt:       clt,
		namespace: namespace,
	}
}

func (query *DynakubeQuery) Proxy(dynakube *dynatracev1beta1.DynaKube) (string, error) {
	clt := query.clt
	namespace := query.namespace

	if dynakube.Spec.Proxy != nil {
		if dynakube.Spec.Proxy.ValueFrom != "" {
			var proxySecret corev1.Secret

			if err := clt.Get(context.TODO(), client.ObjectKey{Name: dynakube.Spec.Proxy.ValueFrom, Namespace: namespace}, &proxySecret); err != nil {
				return "", errors.WithMessage(err, "failed to query proxy")
			}

			return string(proxySecret.Data[proxyKey]), nil
		} else if dynakube.Spec.Proxy.Value != "" {
			return dynakube.Spec.Proxy.Value, nil
		}
	}

	return "", nil
}

func (query *DynakubeQuery) TrustedCAs(dynakube *dynatracev1beta1.DynaKube) ([]byte, error) {
	clt := query.clt
	namespace := query.namespace

	if dynakube.Spec.TrustedCAs != "" {
		var caConfigMap corev1.ConfigMap

		if err := clt.Get(context.TODO(), client.ObjectKey{Name: dynakube.Spec.TrustedCAs, Namespace: namespace}, &caConfigMap); err != nil {
			return nil, errors.WithMessage(err, "failed to query ca")
		}

		return []byte(caConfigMap.Data[trustedCAKey]), nil
	}

	return nil, nil
}

func (query *DynakubeQuery) TlsCert(dynakube *dynatracev1beta1.DynaKube) (string, error) {
	clt := query.clt
	namespace := query.namespace

	if dynakube.HasActiveGateCaCert() {
		var tlsSecret corev1.Secret

		if err := clt.Get(context.TODO(), client.ObjectKey{Name: dynakube.Spec.ActiveGate.TlsSecretName, Namespace: namespace}, &tlsSecret); err != nil {
			return "", errors.WithMessage(err, "failed to query tls secret")
		}

		return string(tlsSecret.Data[tlsCertKey]), nil
	}

	return "", nil
}
