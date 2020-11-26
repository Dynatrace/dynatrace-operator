package dtclient

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToken_Contains(t *testing.T) {
	tokenscopes := TokenScopes{}
	existingScope := "test-scope"

	tokenscopes = append(tokenscopes, existingScope)

	assert.True(t, tokenscopes.Contains(existingScope))
	assert.False(t, tokenscopes.Contains("invalid-scope"))
}

func testGetTokenScopes(t *testing.T, dynatraceClient Client) {
	{
		scopes, err := dynatraceClient.GetTokenScopes("good-token")
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"DataExport", "LogExport"}, scopes)
	}
	{
		scopes, err := dynatraceClient.GetTokenScopes("bad-token")
		assert.Nil(t, scopes)
		assert.Error(t, err)
		assert.Exactly(t, ServerError{Code: 401, Message: "error received from server"}, err)
	}
}

func handleTokenScopes(request *http.Request, writer http.ResponseWriter) {
	var model struct {
		Token string `json:"token"`
	}

	defer func() {
		//Swallow error, nothing can be done at this point
		_ = request.Body.Close()
	}()
	d, _ := ioutil.ReadAll(request.Body)
	err := json.Unmarshal(d, &model)
	if err != nil {
		writeError(writer, http.StatusInternalServerError)
		return
	}

	if request.Method != "POST" {
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
