package core

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type apiModel struct {
	Foo string `json:"foo"`
}

type brokenModel struct {
	A string
}

func (m brokenModel) MarshalJSON() ([]byte, error) {
	return []byte("{]"), nil
}

func TestClient_Verbs(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/"+r.Method, r.URL.Path)
	}))
	defer s.Close()

	c := NewClient(Config{BaseURL: must(url.Parse(s.URL)).JoinPath("/api/")})
	require.NoError(t, c.GET(t.Context(), http.MethodGet).Execute(nil))
	require.NoError(t, c.POST(t.Context(), http.MethodPost).Execute(nil))
	require.NoError(t, c.PUT(t.Context(), http.MethodPut).Execute(nil))
}

func TestClient_Headers(t *testing.T) {
	var expectContentType string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/test", r.URL.Path)
		assert.Equal(t, "my-user-agent", r.UserAgent())
		assert.Equal(t, "application/json", r.Header.Get("accept"))
		assert.Equal(t, expectContentType, r.Header.Get("content-type"))
	}))
	defer s.Close()

	c := NewClient(Config{BaseURL: must(url.Parse(s.URL)).JoinPath("/api/"), UserAgent: "my-user-agent"})
	require.NoError(t, c.GET(t.Context(), "/test").Execute(nil))

	expectContentType = "application/json"
	require.NoError(t, c.POST(t.Context(), "/test").WithRawBody([]byte("test")).Execute(nil))
}

func TestClient_URL(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "a=b&c=d", r.URL.Query().Encode())
	}))
	defer s.Close()

	c := NewClient(Config{BaseURL: must(url.Parse(s.URL))})
	err := c.POST(t.Context(), "/test").
		WithQueryParams(map[string]string{"a": "b", "c": "d"}).
		Execute(nil)
	require.NoError(t, err)
}

func TestClient_Errors(t *testing.T) {
	t.Run("missing base URL", func(t *testing.T) {
		c := new(Client)
		assert.EqualError(t, c.GET(t.Context(), "/test").Execute(nil), "build URL: missing base URL")
	})

	t.Run("invalid json body", func(t *testing.T) {
		c := NewClient(Config{BaseURL: must(url.Parse("http://foo.bar/api")), HTTPClient: &http.Client{}})
		assert.Error(t, c.GET(t.Context(), "/test").WithJSONBody(brokenModel{}).Execute(nil))
	})
}

func TestClient_TokenTypes(t *testing.T) {
	var expectToken string

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), apiTokenHeader)
		assert.Equal(t, expectToken, token)
	}))
	defer s.Close()

	c := NewClient(Config{
		BaseURL:         must(url.Parse(s.URL)),
		APIToken:        "api",
		PaasToken:       "paas",
		DataIngestToken: "data-ingest",
	})

	t.Run("default", func(t *testing.T) {
		expectToken = "api"
		assert.NoError(t, c.GET(t.Context(), "/test").Execute(nil))
	})

	t.Run("paas", func(t *testing.T) {
		expectToken = "paas"
		assert.NoError(t, c.GET(t.Context(), "/test").WithPaasToken().Execute(nil))
	})

	t.Run("data ingest", func(t *testing.T) {
		expectToken = "data-ingest"
		assert.NoError(t, c.GET(t.Context(), "/test").WithTokenType(TokenTypeDataIngest).Execute(nil))
	})
}

func TestClient_Execute(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte(`{"error":{}}`))

			return
		}
		_, _ = w.Write([]byte(`{"foo":"bar"}`))
	}))
	defer s.Close()

	c := NewClient(Config{BaseURL: must(url.Parse(s.URL))})

	t.Run("ok", func(t *testing.T) {
		var model apiModel
		require.NoError(t, c.GET(t.Context(), "/test").Execute(&model))
		assert.Equal(t, "bar", model.Foo)
	})

	t.Run("fail", func(t *testing.T) {
		var model apiModel
		err := c.GET(t.Context(), "/fail").Execute(&model)
		require.Error(t, err)
		assert.Empty(t, model.Foo)
	})
}

func TestClient_ExecuteRaw(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte(`{"error":{}}`))

			return
		}
		_, _ = w.Write([]byte("response"))
	}))
	defer s.Close()

	c := NewClient(Config{BaseURL: must(url.Parse(s.URL))})

	t.Run("ok", func(t *testing.T) {
		body, err := c.GET(t.Context(), "/test").ExecuteRaw()
		require.NoError(t, err)
		assert.Equal(t, "response", string(body))
	})

	t.Run("fail", func(t *testing.T) {
		body, err := c.GET(t.Context(), "/fail").ExecuteRaw()
		require.Error(t, err)
		assert.JSONEq(t, `{"error":{}}`, string(body))
	})
}

func TestHandleErrorResponse_SingleServerError(t *testing.T) {
	resp := newTestResponse(400, "/test", `{"error":{"code":400,"message":"bad request"}}`)
	err := handleErrorResponse(resp, []byte(`{"error":{"code":400,"message":"bad request"}}`))
	httpErr := &HTTPError{}
	require.ErrorAs(t, err, &httpErr)
	require.Len(t, httpErr.ServerErrors, 1)
	assert.Equal(t, 400, httpErr.ServerErrors[0].Code)
	assert.EqualError(t, err, "HTTP 400: dynatrace server error 400: bad request")
}

func TestHandleErrorResponse_MultipleServerErrors(t *testing.T) {
	resp := newTestResponse(400, "/test", `[{"error":{"code":400,"message":"bad1"}},{"error":{"code":400,"message":"bad2"}}]`)
	err := handleErrorResponse(resp, []byte(`[{"error":{"code":400,"message":"bad1"}},{"error":{"code":400,"message":"bad2"}}]`))
	httpErr := &HTTPError{}
	require.ErrorAs(t, err, &httpErr)
	require.Len(t, httpErr.ServerErrors, 2)
	assert.EqualError(t, err, "HTTP 400: dynatrace server error 400: bad1; dynatrace server error 400: bad2")
}

func TestHandleErrorResponse_GenericHTTPError(t *testing.T) {
	htmlBody := `<html><head><title>504 Gateway error</title></head><body><p>Oops!</p></body></html>`
	resp := newTestResponse(500, "/test", "")
	err := handleErrorResponse(resp, []byte(htmlBody))
	httpErr := &HTTPError{}
	require.ErrorAs(t, err, &httpErr)
	assert.Empty(t, httpErr.ServerErrors)
	assert.EqualError(t, err, "HTTP request (/test) failed 500")
}

func newTestResponse(status int, path, body string) *http.Response {
	u := new(url.URL)
	u.Path = path

	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Request: &http.Request{
			URL: u,
		},
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}
