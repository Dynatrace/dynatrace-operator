package bootstrapperconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MetricsURLSecretField   = "DT_METRICS_INGEST_URL"
	MetricsTokenSecretField = "DT_METRICS_INGEST_API_TOKEN"
	configFile              = "endpoint.properties"
)

func (s *SecretGenerator) prepareEndpoints(ctx context.Context, dk *dynakube.DynaKube) (string, error) {
	fields, err := s.prepareFieldsForEndpoints(ctx, dk)
	if err != nil {
		return "", errors.WithStack(err)
	}

	endpointPropertiesBuilder := strings.Builder{}

	if _, err := endpointPropertiesBuilder.WriteString(fmt.Sprintf("%s=%s\n", MetricsURLSecretField, fields[MetricsURLSecretField])); err != nil {
		k8sconditions.SetSecretGenFailed(dk.Conditions(), ConfigConditionType, err)

		return "", errors.WithStack(err)
	}

	if _, err := endpointPropertiesBuilder.WriteString(fmt.Sprintf("%s=%s\n", MetricsTokenSecretField, fields[MetricsTokenSecretField])); err != nil {
		k8sconditions.SetSecretGenFailed(dk.Conditions(), ConfigConditionType, err)

		return "", errors.WithStack(err)
	}

	return endpointPropertiesBuilder.String(), nil
}

func (s *SecretGenerator) prepareFieldsForEndpoints(ctx context.Context, dk *dynakube.DynaKube) (map[string]string, error) {
	fields := make(map[string]string)

	tokens, err := s.secrets.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: dk.Namespace})
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), ConfigConditionType, err)

		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	if token, ok := tokens.Data[dtclient.DataIngestToken]; ok {
		fields[MetricsTokenSecretField] = string(token)
	} else {
		log.Info("data ingest token not found in secret", "dk", dk.Name)
	}

	if ingestURL, err := ingestURLFor(dk); err != nil {
		return nil, err
	} else {
		fields[MetricsURLSecretField] = ingestURL
	}

	return fields, nil
}

func ingestURLFor(dk *dynakube.DynaKube) (string, error) {
	switch {
	case dk.ActiveGate().IsMetricsIngestEnabled():
		return metricsIngestURLForClusterActiveGate(dk)
	case len(dk.Spec.APIURL) > 0:
		return metricsIngestURLForDynatraceActiveGate(dk)
	default:
		return "", errors.New("failed to create metadata-enrichment endpoint, DynaKube.spec.apiUrl is empty")
	}
}

func metricsIngestURLForDynatraceActiveGate(dk *dynakube.DynaKube) (string, error) {
	return dk.Spec.APIURL + "/v2/metrics/ingest", nil
}

func metricsIngestURLForClusterActiveGate(dk *dynakube.DynaKube) (string, error) {
	tenant, err := dk.TenantUUID()
	if err != nil {
		return "", err
	}

	serviceName := capability.BuildServiceName(dk.Name)

	return fmt.Sprintf("http://%s.%s/e/%s/api/v2/metrics/ingest", serviceName, dk.Namespace, tenant), nil
}
