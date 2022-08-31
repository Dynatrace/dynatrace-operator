package capability

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) createOrUpdateEecConfigMap() (bool, error) {
	desired, err := CreateEecConfigMap(r.dynakube, r.capability.ShortName())
	if err != nil {
		return false, errors.WithStack(err)
	}

	installed := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), kubeobjects.Key(desired), installed)
	if k8serrors.IsNotFound(err) {
		log.Info("creating EEC config map", "module", r.capability.ShortName())
		if err = controllerutil.SetControllerReference(r.dynakube, desired, r.client.Scheme()); err != nil {
			return false, errors.WithStack(err)
		}

		err = r.client.Create(context.TODO(), desired)
		return true, errors.WithStack(err)
	}

	if err == nil {
		if !kubeobjects.ConfigMapDataEqual(installed, desired) {
			desired.ObjectMeta.ResourceVersion = installed.ObjectMeta.ResourceVersion

			if err = r.client.Update(context.TODO(), desired); err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, errors.WithStack(err)
}
