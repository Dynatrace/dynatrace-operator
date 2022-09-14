package customproperties

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
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

var _ kubeobjects.Reconciler = &Reconciler{}

type Reconciler struct {
	client                    client.Client
	scheme                    *runtime.Scheme
	customPropertiesSource    *dynatracev1beta1.DynaKubeValueSource
	customPropertiesOwnerName string
	instance                  *dynatracev1beta1.DynaKube
}

func NewReconciler(clt client.Client, instance *dynatracev1beta1.DynaKube, customPropertiesOwnerName string, scheme *runtime.Scheme, customPropertiesSource *dynatracev1beta1.DynaKubeValueSource) *Reconciler {
	return &Reconciler{
		client:                    clt,
		instance:                  instance,
		scheme:                    scheme,
		customPropertiesSource:    customPropertiesSource,
		customPropertiesOwnerName: customPropertiesOwnerName,
	}
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	if r.customPropertiesSource == nil {
		return false, nil
	}

	if r.hasCustomPropertiesValueOnly() {
		mustNotUpdate, err := r.createCustomPropertiesIfNotExists()
		if err != nil {
			log.Error(err, "could not create custom properties", "owner", r.customPropertiesOwnerName)
			return false, errors.WithStack(err)
		}

		if !mustNotUpdate {
			err = r.updateCustomPropertiesIfOutdated()
			if err != nil {
				log.Error(err, "could not update custom properties", "owner", r.customPropertiesOwnerName)
				return false, errors.WithStack(err)
			}
		}
	}

	return false, nil
}

func (r *Reconciler) createCustomPropertiesIfNotExists() (bool, error) {
	var customPropertiesSecret corev1.Secret
	err := r.client.Get(context.TODO(),
		client.ObjectKey{Name: r.buildCustomPropertiesName(r.instance.Name), Namespace: r.instance.Namespace}, &customPropertiesSecret)
	if err != nil && k8serrors.IsNotFound(err) {
		return true, r.createCustomProperties()
	}
	return false, errors.WithStack(err)
}

func (r *Reconciler) updateCustomPropertiesIfOutdated() error {
	var customPropertiesSecret corev1.Secret
	err := r.client.Get(context.TODO(),
		client.ObjectKey{Name: r.buildCustomPropertiesName(r.instance.Name), Namespace: r.instance.Namespace},
		&customPropertiesSecret)
	if err != nil {
		return errors.WithStack(err)
	}
	if r.isOutdated(&customPropertiesSecret) {
		return r.updateCustomProperties(&customPropertiesSecret)
	}
	return nil
}

func (r *Reconciler) isOutdated(customProperties *corev1.Secret) bool {
	return r.customPropertiesSource.Value != string(customProperties.Data[DataKey])
}

func (r *Reconciler) updateCustomProperties(customProperties *corev1.Secret) error {
	customProperties.Data[DataKey] = []byte(r.customPropertiesSource.Value)
	return r.client.Update(context.TODO(), customProperties)
}

func (r *Reconciler) createCustomProperties() error {
	customPropertiesSecret := r.buildCustomPropertiesSecret(
		r.buildCustomPropertiesName(r.instance.Name),
		r.customPropertiesSource.Value,
	)

	err := controllerutil.SetControllerReference(r.instance, customPropertiesSecret, r.scheme)
	if err != nil {
		return errors.WithStack(err)
	}

	return r.client.Create(context.TODO(), customPropertiesSecret)
}

func (r *Reconciler) buildCustomPropertiesSecret(secretName string, data string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: r.instance.Namespace,
		},
		Data: map[string][]byte{
			DataKey: []byte(data),
		},
	}
}

func (r *Reconciler) buildCustomPropertiesName(name string) string {
	return fmt.Sprintf("%s-%s-%s", name, r.customPropertiesOwnerName, Suffix)
}

func (r *Reconciler) hasCustomPropertiesValueOnly() bool {
	return r.customPropertiesSource.Value != "" &&
		r.customPropertiesSource.ValueFrom == ""
}
