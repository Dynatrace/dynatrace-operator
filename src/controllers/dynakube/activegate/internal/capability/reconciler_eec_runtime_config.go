package capability

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) createOrUpdateEecConfigMap() error {
	desired, err := CreateEecConfigMap(r.dynakube, r.capability.ShortName())
	if err != nil {
		return errors.WithStack(err)
	}

	installed := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), kubeobjects.Key(desired), installed)

	if k8serrors.IsNotFound(err) {
		log.Info("creating EEC config map", "module", r.capability.ShortName())

		err = controllerutil.SetControllerReference(r.dynakube, desired, r.client.Scheme())
		if err != nil {
			return errors.WithStack(err)
		}

		err = r.client.Create(context.TODO(), desired)
		return errors.WithStack(err)
	}

	if err != nil {
		return errors.WithStack(err)
	}

	if !kubeobjects.ConfigMapDataEqual(installed, desired) {
		desired.ObjectMeta.ResourceVersion = installed.ObjectMeta.ResourceVersion
		err = r.client.Update(context.TODO(), desired)

		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
