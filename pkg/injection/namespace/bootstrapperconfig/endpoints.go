package bootstrapperconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *SecretGenerator) prepareEndpoints(ctx context.Context, dk *dynakube.DynaKube) (string, error) {
	fields, err := s.prepareFieldsForEndpoints(ctx, dk)
	if err != nil {
		return "", errors.WithStack(err)
	}

	endpointPropertiesBuilder := strings.Builder{}

	if _, err := endpointPropertiesBuilder.WriteString(fmt.Sprintf("%s=%s\n", dtingestendpoint.MetricsUrlSecretField, fields[dtingestendpoint.MetricsUrlSecretField])); err != nil {
		conditions.SetSecretGenFailed(dk.Conditions(), ConditionType, err)

		return "", errors.WithStack(err)
	}

	if _, err := endpointPropertiesBuilder.WriteString(fmt.Sprintf("%s=%s\n", dtingestendpoint.MetricsTokenSecretField, fields[dtingestendpoint.MetricsTokenSecretField])); err != nil {
		conditions.SetSecretGenFailed(dk.Conditions(), ConditionType, err)

		return "", errors.WithStack(err)
	}

	return endpointPropertiesBuilder.String(), nil
}

func (s *SecretGenerator) prepareFieldsForEndpoints(ctx context.Context, dk *dynakube.DynaKube) (map[string]string, error) {
	fields := make(map[string]string)

	tokens, err := k8ssecret.Query(s.client, s.apiReader, log).Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: dk.Namespace})
	if err != nil {
		conditions.SetKubeApiError(dk.Conditions(), ConditionType, err)

		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	if token, ok := tokens.Data[dtclient.DataIngestToken]; ok {
		fields[dtingestendpoint.MetricsTokenSecretField] = string(token)
	} else {
		log.Info("data ingest token not found in secret", "dk", dk.Name)
	}

	if ingestUrl, err := dtingestendpoint.IngestUrlFor(dk); err != nil {
		return nil, err
	} else {
		fields[dtingestendpoint.MetricsUrlSecretField] = ingestUrl
	}

	return fields, nil
}
