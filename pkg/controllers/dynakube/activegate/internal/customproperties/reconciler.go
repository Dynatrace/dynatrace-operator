package customproperties

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
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
	customPropertiesSource    *dynakube.DynaKubeValueSource
	dk                        *dynakube.DynaKube
	customPropertiesOwnerName string
}

func NewReconciler(clt client.Client, dk *dynakube.DynaKube, customPropertiesOwnerName string, customPropertiesSource *dynakube.DynaKubeValueSource) *Reconciler {
	return &Reconciler{
		client:                    clt,
		dk:                        dk,
		customPropertiesSource:    customPropertiesSource,
		customPropertiesOwnerName: customPropertiesOwnerName,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.customPropertiesSource == nil {
		return nil
	}

	if r.hasCustomPropertiesValueOnly() {
		customPropertiesSecret, err := secret.Build(r.dk,
			r.buildCustomPropertiesName(r.dk.Name),
			map[string][]byte{
				DataKey: []byte(r.customPropertiesSource.Value),
			},
		)
		if err != nil {
			return err
		}

		_, err = secret.Query(r.client, r.client, log).WithOwner(r.dk).CreateOrUpdate(ctx, customPropertiesSecret) // TODO: pass in an apiReader instead of the client 2 times
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) buildCustomPropertiesName(name string) string {
	return fmt.Sprintf("%s-%s-%s", name, r.customPropertiesOwnerName, Suffix)
}

func (r *Reconciler) hasCustomPropertiesValueOnly() bool {
	return r.customPropertiesSource.Value != "" &&
		r.customPropertiesSource.ValueFrom == ""
}
