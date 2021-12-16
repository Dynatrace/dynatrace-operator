package customproperties

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	Suffix     = "custom-properties"
	DataKey    = "customProperties"
	DataPath   = "custom.properties"
	VolumeName = "custom-properties"
	MountPath  = "/var/lib/dynatrace/gateway/config_template/custom.properties"
)

type CustomPropertiesReconciler struct {
	client.Client
	scheme                    *runtime.Scheme
	customPropertiesSource    dynatracev1beta1.DynaKubeValueSource
	customPropertiesOwnerName string
	instance                  *dynatracev1beta1.DynaKube
}

func NewCustomPropertiesReconciler(clt client.Client, instance *dynatracev1beta1.DynaKube, customPropertiesOwnerName string, customPropertiesSource dynatracev1beta1.DynaKubeValueSource, scheme *runtime.Scheme) *CustomPropertiesReconciler {
	return &CustomPropertiesReconciler{
		Client:                    clt,
		instance:                  instance,
		scheme:                    scheme,
		customPropertiesSource:    customPropertiesSource,
		customPropertiesOwnerName: customPropertiesOwnerName,
	}
}

func (reconciler *CustomPropertiesReconciler) Reconcile() error {
	if reconciler.hasCustomPropertiesValueOnly() {
		mustNotUpdate, err := reconciler.createCustomPropertiesIfNotExists()
		if err != nil {
			log.Error(err, "could not create custom properties", "owner", reconciler.customPropertiesOwnerName)
			return errors.WithStack(err)
		}

		if !mustNotUpdate {
			err = reconciler.updateCustomPropertiesIfOutdated()
			if err != nil {
				log.Error(err, "could not update custom properties", "owner", reconciler.customPropertiesOwnerName)
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

func (reconciler *CustomPropertiesReconciler) createCustomPropertiesIfNotExists() (bool, error) {
	var customPropertiesSecret corev1.Secret
	err := reconciler.Get(context.TODO(),
		client.ObjectKey{Name: reconciler.buildCustomPropertiesName(reconciler.instance.Name), Namespace: reconciler.instance.Namespace}, &customPropertiesSecret)
	if err != nil && k8serrors.IsNotFound(err) {
		return true, reconciler.createCustomProperties()
	}
	return false, errors.WithStack(err)
}

func (reconciler *CustomPropertiesReconciler) updateCustomPropertiesIfOutdated() error {
	var customPropertiesSecret corev1.Secret
	err := reconciler.Get(context.TODO(),
		client.ObjectKey{Name: reconciler.buildCustomPropertiesName(reconciler.instance.Name), Namespace: reconciler.instance.Namespace},
		&customPropertiesSecret)
	if err != nil {
		return errors.WithStack(err)
	}
	if reconciler.isOutdated(&customPropertiesSecret) {
		return reconciler.updateCustomProperties(&customPropertiesSecret)
	}
	return nil
}

func (reconciler *CustomPropertiesReconciler) isOutdated(customProperties *corev1.Secret) bool {
	return reconciler.customPropertiesSource.Value != string(customProperties.Data[DataKey])
}

func (reconciler *CustomPropertiesReconciler) updateCustomProperties(customProperties *corev1.Secret) error {
	customProperties.Data[DataKey] = []byte(reconciler.customPropertiesSource.Value)
	return reconciler.Update(context.TODO(), customProperties)
}

func (reconciler *CustomPropertiesReconciler) createCustomProperties() error {
	customPropertiesSecret := reconciler.buildCustomPropertiesSecret(
		reconciler.buildCustomPropertiesName(reconciler.instance.Name),
		reconciler.customPropertiesSource.Value,
	)

	err := controllerutil.SetControllerReference(reconciler.instance, customPropertiesSecret, reconciler.scheme)
	if err != nil {
		return errors.WithStack(err)
	}

	return reconciler.Create(context.TODO(), customPropertiesSecret)
}

func (reconciler *CustomPropertiesReconciler) buildCustomPropertiesSecret(secretName string, data string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: reconciler.instance.Namespace,
		},
		Data: map[string][]byte{
			DataKey: []byte(data),
		},
	}
}

func (reconciler *CustomPropertiesReconciler) buildCustomPropertiesName(name string) string {
	return fmt.Sprintf("%s-%s-%s", name, reconciler.customPropertiesOwnerName, Suffix)
}

func (reconciler *CustomPropertiesReconciler) hasCustomPropertiesValueOnly() bool {
	return reconciler.customPropertiesSource.Value != "" &&
		reconciler.customPropertiesSource.ValueFrom == ""
}
