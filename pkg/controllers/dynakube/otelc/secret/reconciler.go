package secret

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
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
		return r.ensureTelemetryIngestApiCredentialsSecret(ctx)
	}

	return r.removeTelemetryIngestApiCredentialsSecret(ctx)
}

func (r *Reconciler) ensureTelemetryIngestApiCredentialsSecret(ctx context.Context) error {
	query := k8ssecret.Query(r.client, r.apiReader, log)
	_, err := query.Get(ctx, types.NamespacedName{Name: consts.TelemetryApiCredentialsSecretName, Namespace: r.dk.Namespace})

	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("creating new secret for telemetry api credentials")

		secretConfig, err := r.generateTelemetryIngestApiCredentialsSecret(consts.TelemetryApiCredentialsSecretName)

		if err != nil {
			conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, err)

			return err
		}

		_, err = hasher.GenerateHash(secretConfig.Data)
		if err != nil {
			conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, err)

			return err
		}

		err = query.Create(ctx, secretConfig)
		if err != nil {
			log.Info("could not create secret for telemetry api credentials", "name", secretConfig.Name)
			conditions.SetKubeApiError(r.dk.Conditions(), secretConditionType, err)

			return err
		}

		conditions.SetSecretCreated(r.dk.Conditions(), secretConditionType, consts.TelemetryApiCredentialsSecretName)
	} else if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), secretConditionType, err)

		return err
	}

	return nil
}

func (r *Reconciler) getDtEndpoint() ([]byte, error) {
	if r.dk.ActiveGate().IsEnabled() {
		tenantUUID, err := r.dk.TenantUUID()
		if err != nil {
			return nil, err
		}

		return []byte(fmt.Sprintf("https://%s-activegate.dynatrace.svc/e/%s/api/v2/otlp", r.dk.Name, tenantUUID)), nil
	}

	return []byte(r.dk.ApiUrl() + "/v2/otlp"), nil
}

func (r *Reconciler) generateTelemetryIngestApiCredentialsSecret(name string) (secret *corev1.Secret, err error) {
	secretData := make(map[string][]byte)

	dtEndpoint, err := r.getDtEndpoint()
	if err != nil {
		return nil, err
	}

	secretData["DT_ENDPOINT"] = dtEndpoint

	secretConfig, err := k8ssecret.Build(r.dk,
		name,
		secretData,
		k8ssecret.SetLabels(k8slabels.NewCoreLabels(r.dk.Name, k8slabels.OtelCComponentLabel).BuildLabels()),
	)

	if err != nil {
		return nil, err
	}

	return secretConfig, nil
}

func (r *Reconciler) removeTelemetryIngestApiCredentialsSecret(ctx context.Context) error {
	if meta.FindStatusCondition(*r.dk.Conditions(), secretConditionType) == nil {
		return nil // no condition == nothing is there to clean up
	}

	query := k8ssecret.Query(r.client, r.apiReader, log)
	err := query.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: consts.TelemetryApiCredentialsSecretName, Namespace: r.dk.Namespace}})

	if err != nil {
		log.Error(err, "could not delete apiCredential secret", "name", consts.TelemetryApiCredentialsSecretName)
	}

	meta.RemoveStatusCondition(r.dk.Conditions(), secretConditionType)

	return nil
}
