package bootstrapperconfig

import (
	"context"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/ca"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/pkg/errors"
)

// generate gets the necessary info the create the init secret data
func (s *SecretGenerator) generateCerts(ctx context.Context, dk *dynakube.DynaKube) (map[string][]byte, error) {
	data := map[string][]byte{}

	agCerts, err := dk.ActiveGateTLSCert(ctx, s.apiReader)
	if err != nil {
		conditions.SetKubeApiError(dk.Conditions(), ConditionType, err)

		return nil, errors.WithStack(err)
	}

	if len(agCerts) != 0 {
		data[ca.AgCertsInputFile] = agCerts
	}

	trustedCAs, err := dk.TrustedCAs(ctx, s.apiReader)

	if len(trustedCAs) != 0 {
		data[ca.TrustedCertsInputFile] = trustedCAs
	}

	return data, err
}
