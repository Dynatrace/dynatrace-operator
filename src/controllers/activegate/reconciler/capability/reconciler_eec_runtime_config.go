package capability

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) createEecConfigMapIfNotExists() (bool, error) {
	eecConfigMap := CreateEecConfigMap(r.Instance, r.ShortName())

	getErr := r.Get(context.TODO(), client.ObjectKey{Name: eecConfigMap.Name, Namespace: eecConfigMap.Namespace}, eecConfigMap)
	if getErr != nil && k8serrors.IsNotFound(getErr) {
		log.Info("creating EEC config map", "module", r.ShortName())
		if err := controllerutil.SetControllerReference(r.Instance, eecConfigMap, r.Scheme()); err != nil {
			return false, errors.WithStack(err)
		}

		err := r.Create(context.TODO(), eecConfigMap)
		return true, errors.WithStack(err)
	}
	return false, errors.WithStack(getErr)
}

func (r *Reconciler) updateEecConfigMapIfOutdated() (bool, error) {
	desiredConfigMap := CreateEecConfigMap(r.Instance, r.ShortName())
	installedConfigMap := &corev1.ConfigMap{}

	err := r.Get(context.TODO(), client.ObjectKey{Name: desiredConfigMap.Name, Namespace: desiredConfigMap.Namespace}, installedConfigMap)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if r.isEecConfigMapOutdated(installedConfigMap, desiredConfigMap) {
		desiredConfigMap.ObjectMeta.ResourceVersion = installedConfigMap.ObjectMeta.ResourceVersion
		updateErr := r.updateEecConfigMap(desiredConfigMap)
		if updateErr != nil {
			return false, updateErr
		}
		return true, nil
	}
	return false, nil
}

func (r *Reconciler) isEecConfigMapOutdated(installedConfigMap, desiredConfigMap *corev1.ConfigMap) bool {
	configMapsEqual := reflect.DeepEqual(installedConfigMap.Data, desiredConfigMap.Data) &&
		reflect.DeepEqual(installedConfigMap.BinaryData, desiredConfigMap.BinaryData)
	return !configMapsEqual
}

func (r *Reconciler) updateEecConfigMap(eecConfigMap *corev1.ConfigMap) error {
	return r.Update(context.TODO(), eecConfigMap)
}
