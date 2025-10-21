package endpoint

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8sconfigmap "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/configmap"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	dk         *dynakube.DynaKube
	configMaps k8sconfigmap.QueryObject
}

type ReconcilerBuilder func(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler

func NewReconciler(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		dk:         dk,
		configMaps: k8sconfigmap.Query(client, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.TelemetryIngest().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), configMapConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		defer meta.RemoveStatusCondition(r.dk.Conditions(), configMapConditionType)

		configMap, _ := k8sconfigmap.Build(r.dk,
			consts.OtlpAPIEndpointConfigMapName,
			nil,
		)

		_ = r.deleteConfigMap(ctx, configMap)

		return nil
	}

	return r.reconcileConfigMap(ctx)
}

func (r *Reconciler) deleteConfigMap(ctx context.Context, configMap *corev1.ConfigMap) error {
	log.Info("deleting configmap", "name", configMap.Name)

	err := r.configMaps.Delete(ctx, configMap)

	if err != nil && !k8serrors.IsNotFound(err) {
		conditions.SetKubeAPIError(r.dk.Conditions(), configMapConditionType, err)

		return errors.WithMessagef(err, "failed to delete configMap %s", configMap.Name)
	}

	return nil
}

func (r *Reconciler) reconcileConfigMap(ctx context.Context) error {
	configMapData, err := r.generateData()
	if err != nil {
		return errors.WithMessage(err, "could not generate config map data")
	}

	configMap, err := k8sconfigmap.Build(r.dk,
		consts.OtlpAPIEndpointConfigMapName,
		configMapData,
		k8sconfigmap.SetLabels(k8slabels.NewCoreLabels(r.dk.Name, k8slabels.OtelCComponentLabel).BuildLabels()),
	)
	if err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), configMapConditionType, err)

		return errors.WithStack(err)
	}

	_, err = r.configMaps.CreateOrUpdate(ctx, configMap)
	if err != nil {
		log.Info("could not create or update config map", "name", configMap.Name)
		conditions.SetKubeAPIError(r.dk.Conditions(), configMapConditionType, errors.WithMessage(err, "failed to create or update config map"))

		return errors.WithMessage(err, "failed to create or update config map")
	}

	conditions.SetConfigMapCreatedOrUpdated(r.dk.Conditions(), configMapConditionType, configMap.Name)

	return nil
}

func (r *Reconciler) generateData() (map[string]string, error) {
	data := make(map[string]string)

	dtEndpoint, err := BuildOTLPEndpoint(*r.dk)
	if err != nil {
		return data, err
	}

	data["DT_ENDPOINT"] = dtEndpoint

	return data, nil
}

func BuildOTLPEndpoint(dk dynakube.DynaKube) (string, error) {
	dtEndpoint := dk.APIURL() + "/v2/otlp"

	if dk.ActiveGate().IsEnabled() {
		tenantUUID, err := dk.TenantUUID()
		if err != nil {
			return "", err
		}

		serviceFQDN := capability.BuildServiceName(dk.Name) + "." + dk.Namespace + ".svc"

		dtEndpoint = fmt.Sprintf("https://%s/e/%s/api/v2/otlp", serviceFQDN, tenantUUID)
	}

	return dtEndpoint, nil
}
