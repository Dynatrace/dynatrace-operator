package secret

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const telemetryApiCredentialsSecretName = "dynatrace-telemetry-api-credentials"

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
	return ensureOpenSignalAPISecret(ctx, r.client, r.apiReader, r.dk)
}

func ensureOpenSignalAPISecret(ctx context.Context, client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) error {
	query := k8ssecret.Query(client, apiReader, log)
	_, err := query.Get(ctx, types.NamespacedName{Name: telemetryApiCredentialsSecretName, Namespace: dk.Namespace})

	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("creating new secret for telemetry api credentials")

		secretConfig, err := generateTelemetryApiCredentialsSecret(telemetryApiCredentialsSecretName, dk)

		if err != nil {
			conditions.SetSecretGenFailed(dk.Conditions(), secretConditionType, err)

			return err
		}

		_, err = hasher.GenerateHash(secretConfig.Data)
		if err != nil {
			conditions.SetSecretGenFailed(dk.Conditions(), secretConditionType, err)

			return err
		}

		err = query.Create(ctx, secretConfig)
		if err != nil {
			log.Info("could not create secret for telemetry api credentials", "name", secretConfig.Name)
			conditions.SetKubeApiError(dk.Conditions(), secretConditionType, err)

			return err
		}

		//dk.TokenSecretHash = tokenHash
		conditions.SetSecretCreated(dk.Conditions(), secretConditionType, telemetryApiCredentialsSecretName)
	}

	return nil
}

func generateTelemetryApiCredentialsSecret(name string, dk *dynakube.DynaKube) (secret *corev1.Secret, err error) {
	secretData := make(map[string][]byte)
	// TODO: read api token from secret
	// dk.Tokens()
	secretData["DT_API_TOKEN"] = []byte("")

	tenantUUID, err := dk.TenantUUID()
	if err != nil {
		return nil, err
	}

	if dk.ActiveGate().IsApiEnabled() {
		secretData["DT_ENDPOINT"] = []byte(fmt.Sprintf("https://%s-activegate.dynatrace.svc/e/%s/api/v2/otlp", dk.Name, tenantUUID))
	} else {
		secretData["DT_ENDPOINT"] = []byte(fmt.Sprintf("https://%s.dev.dynatracelabs.com/api/v2/otlp", tenantUUID))
	}

	secretConfig, err := k8ssecret.Build(dk,
		name,
		secretData,
		k8ssecret.SetLabels(k8slabels.NewCoreLabels(dk.Name, k8slabels.OtelCComponentLabel).BuildLabels()),
	)

	if err != nil {
		return nil, err
	}

	return secretConfig, nil
}
