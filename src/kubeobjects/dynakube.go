package kubeobjects

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DynakubeQuery struct {
	kubeReader client.Reader
	namespace  string
	ctx        context.Context
}

func NewDynakubeQuery(kubeReader client.Reader, namespace string) DynakubeQuery {
	return DynakubeQuery{
		kubeReader: kubeReader,
		namespace:  namespace,
		ctx:        nil,
	}
}

func (query DynakubeQuery) Get(objectKey client.ObjectKey) (dynatracev1beta1.DynaKube, error) {
	var dynakube dynatracev1beta1.DynaKube
	err := query.kubeReader.Get(query.ctx, objectKey, &dynakube)

	return dynakube, errors.WithStack(err)
}

func (query DynakubeQuery) List() (dynatracev1beta1.DynaKubeList, error) {
	var dynakubes dynatracev1beta1.DynaKubeList
	err := query.kubeReader.List(query.context(), &dynakubes, client.InNamespace(query.namespace))

	return dynakubes, errors.WithStack(err)
}

func (query DynakubeQuery) WithContext(ctx context.Context) DynakubeQuery {
	query.ctx = ctx

	return query
}

func (query DynakubeQuery) context() context.Context {
	if query.ctx == nil {
		return context.TODO()
	}

	return query.ctx
}

func (query DynakubeQuery) Proxy(dynakube dynatracev1beta1.DynaKube) (string, error) {
	if dynakube.Spec.Proxy != nil {
		if dynakube.Spec.Proxy.ValueFrom != "" {
			var proxySecret corev1.Secret
			err := query.kubeReader.Get(query.context(), client.ObjectKey{Name: dynakube.Spec.Proxy.ValueFrom, Namespace: query.namespace}, &proxySecret)

			if err != nil {
				return "", errors.WithMessage(err, "failed to query proxy")
			}

			return string(proxySecret.Data[dynatracev1beta1.ProxyKey]), nil
		} else if dynakube.Spec.Proxy.Value != "" {
			return dynakube.Spec.Proxy.Value, nil
		}
	}

	return "", nil
}

func (query DynakubeQuery) TrustedCAs(dynakube dynatracev1beta1.DynaKube) ([]byte, error) {
	if dynakube.Spec.TrustedCAs != "" {
		var caConfigMap corev1.ConfigMap
		err := query.kubeReader.Get(query.context(), client.ObjectKey{Name: dynakube.Spec.TrustedCAs, Namespace: query.namespace}, &caConfigMap)

		if err != nil {
			return nil, errors.WithMessage(err, "failed to query ca")
		}

		return []byte(caConfigMap.Data[dynatracev1beta1.TrustedCAKey]), nil
	}

	return nil, nil
}

func (query DynakubeQuery) TlsCert(dynakube dynatracev1beta1.DynaKube) (string, error) {
	if dynakube.HasActiveGateCaCert() {
		var tlsSecret corev1.Secret
		err := query.kubeReader.Get(query.context(), client.ObjectKey{Name: dynakube.Spec.ActiveGate.TlsSecretName, Namespace: query.namespace}, &tlsSecret)

		if err != nil {
			return "", errors.WithMessage(err, "failed to query tls secret")
		}

		return string(tlsSecret.Data[dynatracev1beta1.TlsCertKey]), nil
	}

	return "", nil
}
