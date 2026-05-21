package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
)

const (
	warningPaasTokenNotUsed = "The '" + token.PaaSKey + "' token in the spec.tokens secret is deprecated. It will be ignored because the '" + token.APIKey + "' field in the secret contains a platform token, which will be used for authentication."
	warningPaasTokenUsed    = "The '" + token.PaaSKey + "' token in the spec.tokens secret is deprecated. It will be used for authentication because the '" + token.APIKey + "' field in the secret does not contain a platform token."
)

func deprecatedPaasToken(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	tokenReader := token.NewReader(dv.apiReader, dk)

	tokens, err := tokenReader.ReadTokens(ctx)
	if err != nil {
		log.Info("error occurred while reading the tokens secret", "err", err.Error())

		return ""
	}

	if tokens.PaasToken().Value != "" {
		if dttoken.IsPlatform(tokens.APIToken().Value) {
			return warningPaasTokenNotUsed
		}

		return warningPaasTokenUsed
	}

	return ""
}
