package customproperties

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dynakube"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	Suffix     = "-custom-properties"
	DataKey    = "customProperties"
	DataPath   = "custom.properties"
	VolumeName = "custom-properties"
	MountPath  = "/mnt/dynatrace/gateway/config"
)

type Reconciler struct {
	client.Client
	scheme                    *runtime.Scheme
	log                       logr.Logger
	customPropertiesSource    v1alpha1.DynaKubeValueSource
	customPropertiesOwnerName string
}

func NewReconciler(clt client.Client, log logr.Logger, customPropertiesOwnerName string, customPropertiesSource v1alpha1.DynaKubeValueSource, scheme *runtime.Scheme) *Reconciler {
	return &Reconciler{
		Client:                    clt,
		scheme:                    scheme,
		log:                       log,
		customPropertiesSource:    customPropertiesSource,
		customPropertiesOwnerName: customPropertiesOwnerName,
	}
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	instance, err := dynakube.GetInstance(r, request)
	if err != nil {
		return activegate.LogError(r.log, err, "could not get DynaKube instance")
	}

	if r.hasCustomPropertiesValueOnly() {
		err := r.createCustomPropertiesIfNotExists(instance)
		if err != nil {
			return activegate.LogError(r.log, err, fmt.Sprintf("could not create custom properties for '%s'", r.customPropertiesOwnerName))
		}

		err = r.updateCustomPropertiesIfOutdated(instance)
		if err != nil {
			return activegate.LogError(r.log, err, fmt.Sprintf("could not update custom properties for '%s'", r.customPropertiesOwnerName))
		}

		err = r.setControllerReference(instance)
		if err != nil {
			return activegate.LogError(r.log, err, fmt.Sprintf("could not set controller reference for custom properties of '%s'", r.customPropertiesOwnerName))
		}
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) createCustomPropertiesIfNotExists(instance *v1alpha1.DynaKube) error {
	var customPropertiesSecret *corev1.Secret
	err := r.Get(context.TODO(),
		client.ObjectKey{Name: r.buildCustomPropertiesName(instance.Name), Namespace: activegate.DynatraceNamespace},
		customPropertiesSecret)
	if err != nil && k8serrors.IsNotFound(err) {
		return r.createCustomProperties(instance)
	}
	return err
}

func (r *Reconciler) updateCustomPropertiesIfOutdated(instance *v1alpha1.DynaKube) error {
	var customPropertiesSecret *corev1.Secret
	err := r.Get(context.TODO(),
		client.ObjectKey{Name: r.buildCustomPropertiesName(instance.Name), Namespace: activegate.DynatraceNamespace},
		customPropertiesSecret)
	if err != nil {
		return err
	}
	if r.isOutdated(customPropertiesSecret) {
		return r.updateCustomProperties(customPropertiesSecret)
	}
	return nil
}

func (r *Reconciler) setControllerReference(instance *v1alpha1.DynaKube) error {
	var customPropertiesSecret *corev1.Secret
	err := r.Get(context.TODO(),
		client.ObjectKey{Name: r.buildCustomPropertiesName(instance.Name), Namespace: activegate.DynatraceNamespace},
		customPropertiesSecret)
	if err != nil {
		return err
	}
	return controllerutil.SetControllerReference(instance, customPropertiesSecret, r.scheme)
}

func (r *Reconciler) isOutdated(customProperties *corev1.Secret) bool {
	return r.customPropertiesSource.Value != string(customProperties.Data[DataKey])
}

func (r *Reconciler) updateCustomProperties(customProperties *corev1.Secret) error {
	customProperties.Data[DataKey] = []byte(r.customPropertiesSource.Value)
	return r.Update(context.TODO(), customProperties)
}

func (r *Reconciler) createCustomProperties(instance *v1alpha1.DynaKube) error {
	customPropertiesSecret := buildCustomPropertiesSecret(
		r.buildCustomPropertiesName(instance.Name),
		r.customPropertiesSource.Value,
	)
	return r.Create(context.TODO(), customPropertiesSecret)
}

func buildCustomPropertiesSecret(secretName string, data string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: activegate.DynatraceNamespace,
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
