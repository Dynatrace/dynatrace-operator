package dynatrace

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToken_Contains(t *testing.T) {
	tokenscopes := TokenScopes{}
	existingScope := "test-scope"

	tokenscopes = append(tokenscopes, existingScope)

	assert.True(t, tokenscopes.Contains(existingScope))
	assert.False(t, tokenscopes.Contains("invalid-scope"))
}

func testGetTokenScopes(t *testing.T, dynatraceClient Client) {
	ctx := context.Background()

	t.Run("happy path", func(t *testing.T) {
		scopes, err := dynatraceClient.GetTokenScopes(ctx, "good-token")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"DataExport", "LogExport"}, scopes)
	})

	t.Run("sad path", func(t *testing.T) {
		scopes, err := dynatraceClient.GetTokenScopes(ctx, "bad-token")
		assert.Nil(t, scopes)
		require.Error(t, err)
		assert.Exactly(t, ServerError{Code: 401, Message: "error received from server"}, errors.Cause(err))
	})
}

func handleTokenScopes(request *http.Request, writer http.ResponseWriter) {
	var model struct {
		Token string `json:"token"`
	}

	defer func() {
		// Swallow error, nothing can be done at this point
		_ = request.Body.Close()
	}()

	d, _ := io.ReadAll(request.Body)

	err := json.Unmarshal(d, &model)
	if err != nil {
		writeError(writer, http.StatusInternalServerError)

		return
	}

	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed)

		return
	}

	switch model.Token {
	case "good-token":
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{
			"id": "f7060574-e8cf-4bc2-a9e0-307517ca9957",
			"name": "the-token",
			"userId": "the-user",
			"scopes": [
				"DataExport",
				"LogExport"
			]
		}`))
	default:
		writeError(writer, http.StatusUnauthorized)
	}
}
