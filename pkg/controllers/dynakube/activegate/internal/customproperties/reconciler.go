package customproperties

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Suffix     = "custom-properties"
	DataKey    = "customProperties"
	DataPath   = "custom.properties"
	VolumeName = "custom-properties"
	MountPath  = "/var/lib/dynatrace/gateway/config_template/custom.properties"
)

var _ controllers.Reconciler = &Reconciler{}

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

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.customPropertiesSource == nil {
		return nil
	}

	if r.hasCustomPropertiesValueOnly() {
		mustNotUpdate, err := r.createCustomPropertiesIfNotExists(ctx)
		if err != nil {
			log.Error(err, "could not create custom properties", "owner", r.customPropertiesOwnerName)
			return errors.WithStack(err)
		}

		if !mustNotUpdate {
			err = r.updateCustomPropertiesIfOutdated(ctx)
			if err != nil {
				log.Error(err, "could not update custom properties", "owner", r.customPropertiesOwnerName)
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

func (r *Reconciler) createCustomPropertiesIfNotExists(ctx context.Context) (bool, error) {
	var customPropertiesSecret corev1.Secret

	err := r.client.Get(ctx,
		client.ObjectKey{Name: r.buildCustomPropertiesName(r.instance.Name), Namespace: r.instance.Namespace}, &customPropertiesSecret)
	if err != nil && k8serrors.IsNotFound(err) {
		return true, r.createCustomProperties()
	}

	return false, errors.WithStack(err)
}

func (r *Reconciler) updateCustomPropertiesIfOutdated(ctx context.Context) error {
	var customPropertiesSecret corev1.Secret

	err := r.client.Get(ctx,
		client.ObjectKey{Name: r.buildCustomPropertiesName(r.instance.Name), Namespace: r.instance.Namespace},
		&customPropertiesSecret)
	if err != nil {
		return errors.WithStack(err)
	}

	if r.isOutdated(&customPropertiesSecret) {
		return r.updateCustomProperties(ctx, &customPropertiesSecret)
	}

	return nil
}

func (r *Reconciler) isOutdated(customProperties *corev1.Secret) bool {
	return r.customPropertiesSource.Value != string(customProperties.Data[DataKey])
}

func (r *Reconciler) updateCustomProperties(ctx context.Context, customProperties *corev1.Secret) error {
	customProperties.Data[DataKey] = []byte(r.customPropertiesSource.Value)
	return r.client.Update(ctx, customProperties)
}

func (r *Reconciler) createCustomProperties() error {
	customPropertiesSecret, err := secret.Create(r.scheme, r.instance,
		secret.NewNameModifier(r.buildCustomPropertiesName(r.instance.Name)),
		secret.NewNamespaceModifier(r.instance.Namespace),
		secret.NewDataModifier(map[string][]byte{
			DataKey: []byte(r.customPropertiesSource.Value),
		}))
	if err != nil {
		return errors.WithStack(err)
	}

	return r.client.Create(context.TODO(), customPropertiesSecret)
}

func (r *Reconciler) buildCustomPropertiesName(name string) string {
	return fmt.Sprintf("%s-%s-%s", name, r.customPropertiesOwnerName, Suffix)
}

func (r *Reconciler) hasCustomPropertiesValueOnly() bool {
	return r.customPropertiesSource.Value != "" &&
		r.customPropertiesSource.ValueFrom == ""
}
