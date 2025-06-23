package endpoint

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
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
		return r.ensureOtlpAPIEndpointConfigMap(ctx)
	}

	return r.removeOtlpAPIEndpointConfigMap(ctx)
}

func (r *Reconciler) ensureOtlpAPIEndpointConfigMap(ctx context.Context) error {
	query := k8sconfigmap.Query(r.client, r.apiReader, log)
	_, err := query.Get(ctx, types.NamespacedName{Name: consts.OtlpAPIEndpointConfigMapName, Namespace: r.dk.Namespace})

	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("creating new config map for telemetry api endpoint")

		configMap, err := r.generateOtlpAPIEndpointConfigMap(consts.OtlpAPIEndpointConfigMapName)

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
			conditions.SetKubeAPIError(r.dk.Conditions(), configMapConditionType, err)

			return err
		}

		conditions.SetConfigMapCreatedOrUpdated(r.dk.Conditions(), configMapConditionType, consts.OtlpAPIEndpointConfigMapName)
	} else if err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), configMapConditionType, err)

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

		serviceFQDN := capability.BuildServiceName(r.dk.Name) + "." + r.dk.Namespace + ".svc"

		return fmt.Sprintf("https://%s/e/%s/api/v2/otlp", serviceFQDN, tenantUUID), nil
	}

	return r.dk.APIURL() + "/v2/otlp", nil
}

func (r *Reconciler) generateOtlpAPIEndpointConfigMap(name string) (secret *corev1.ConfigMap, err error) {
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

func (r *Reconciler) removeOtlpAPIEndpointConfigMap(ctx context.Context) error {
	if meta.FindStatusCondition(*r.dk.Conditions(), configMapConditionType) == nil {
		return nil // no condition == nothing is there to clean up
	}

	query := k8sconfigmap.Query(r.client, r.apiReader, log)
	err := query.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: consts.OtlpAPIEndpointConfigMapName, Namespace: r.dk.Namespace}})

	if err != nil {
		log.Error(err, "could not delete apiEndpoint config map", "name", consts.OtlpAPIEndpointConfigMapName)
	}

	meta.RemoveStatusCondition(r.dk.Conditions(), configMapConditionType)

	return nil
}
