package endpoint

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	k8sconfigmap "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/configmap"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dk        *dynakube.DynaKube
}

type ReconcilerBuilder func(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler

func NewReconciler(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		client:    client,
		dk:        dk,
		apiReader: apiReader,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dk.TelemetryIngest().IsEnabled() {
		return r.ensureOtlpApiEndpointConfigMap(ctx)
	}

	return r.removeOtlpApiEndpointConfigMap(ctx)
}

func (r *Reconciler) ensureOtlpApiEndpointConfigMap(ctx context.Context) error {
	query := k8sconfigmap.Query(r.client, r.apiReader, log)
	_, err := query.Get(ctx, types.NamespacedName{Name: consts.OtlpApiEndpointConfigMapName, Namespace: r.dk.Namespace})

	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("creating new config map for telemetry api endpoint")

		configMap, err := r.generateOtlpApiEndpointConfigMap(consts.OtlpApiEndpointConfigMapName)

		if err != nil {
			conditions.SetConfigMapGenFailed(r.dk.Conditions(), configMapConditionType, err)

			return err
		}

		_, err = hasher.GenerateHash(configMap.Data)
		if err != nil {
			conditions.SetConfigMapGenFailed(r.dk.Conditions(), configMapConditionType, err)

			return err
		}

		err = query.Create(ctx, configMap)
		if err != nil {
			log.Info("could not create secret for telemetry api credentials", "name", configMap.Name)
			conditions.SetKubeApiError(r.dk.Conditions(), configMapConditionType, err)

			return err
		}

		conditions.SetConfigMapCreatedOrUpdated(r.dk.Conditions(), configMapConditionType, consts.OtlpApiEndpointConfigMapName)
	} else if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), configMapConditionType, err)

		return err
	}

	return nil
}

func (r *Reconciler) getDtEndpoint() (string, error) {
	if r.dk.ActiveGate().IsEnabled() {
		tenantUUID, err := r.dk.TenantUUID()
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("https://%s-activegate.dynatrace.svc/e/%s/api/v2/otlp", r.dk.Name, tenantUUID), nil
	}

	return r.dk.ApiUrl() + "/v2/otlp", nil
}

func (r *Reconciler) generateOtlpApiEndpointConfigMap(name string) (secret *corev1.ConfigMap, err error) {
	data := make(map[string]string)

	dtEndpoint, err := r.getDtEndpoint()
	if err != nil {
		return nil, err
	}

	data["DT_ENDPOINT"] = dtEndpoint

	configMap, err := k8sconfigmap.Build(r.dk,
		name,
		data,
		k8sconfigmap.SetLabels(k8slabels.NewCoreLabels(r.dk.Name, k8slabels.OtelCComponentLabel).BuildLabels()),
	)

	if err != nil {
		return nil, err
	}

	return configMap, nil
}

func (r *Reconciler) removeOtlpApiEndpointConfigMap(ctx context.Context) error {
	if meta.FindStatusCondition(*r.dk.Conditions(), configMapConditionType) == nil {
		return nil // no condition == nothing is there to clean up
	}

	query := k8sconfigmap.Query(r.client, r.apiReader, log)
	err := query.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: consts.OtlpApiEndpointConfigMapName, Namespace: r.dk.Namespace}})

	if err != nil {
		log.Error(err, "could not delete apiEndpoint config map", "name", consts.OtlpApiEndpointConfigMapName)
	}

	meta.RemoveStatusCondition(r.dk.Conditions(), configMapConditionType)

	return nil
}
