package dynatrace

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

// TokenScopes is a list of scopes assigned to a token
type TokenScopes []string

// Contains returns true if scope is included on the scopes, or false otherwise.
func (s TokenScopes) Contains(scope string) bool {
	return slices.Contains(s, scope)
}

func (dtc *dynatraceClient) GetTokenScopes(ctx context.Context, token string) (TokenScopes, error) {
	var model struct {
		Token string `json:"token"`
	}

	model.Token = token

	jsonStr, err := json.Marshal(model)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dtc.getTokensLookupURL(), bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, errors.WithMessage(err, "error initializing http request")
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", APITokenHeader+token)

	resp, err := dtc.httpClient.Do(req)
	if err != nil {
		return nil, errors.WithMessage(err, "error making post request to dynatrace api")
	}

	defer utils.CloseBodyAfterRequest(resp)

	data, err := dtc.getServerResponseData(resp)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return dtc.readResponseForTokenScopes(data)
}

func (dtc *dynatraceClient) readResponseForTokenScopes(response []byte) (TokenScopes, error) {
	var jr struct {
		Scopes []string `json:"scopes"`
	}

	if err := json.Unmarshal(response, &jr); err != nil {
		log.Error(err, "unable to unmarshal token scopes response", "response", string(response))

		return nil, err
	}

	return jr.Scopes, nil
}
