package core

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type apiModel struct {
	Foo string `json:"foo"`
}

// cacheableModel implements Cacheable: it is considered empty when Foo is unset.
type cacheableModel struct {
	Foo string `json:"foo"`
}

func (m *cacheableModel) IsEmpty() bool { return m.Foo == "" }

type brokenModel struct {
	A string
}

func (m brokenModel) MarshalJSON() ([]byte, error) {
	return []byte("{]"), nil
}

func TestClient_Verbs(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/test/"+r.Method, r.URL.Path)
	}))
	defer s.Close()

	c := NewClient(Config{BaseURL: must(url.Parse(s.URL)).JoinPath("/api/")})
	require.NoError(t, c.GET(t.Context(), "/").WithPath("/test//", http.MethodGet).Execute(nil))
	require.NoError(t, c.POST(t.Context(), "/").WithPath("/test//", http.MethodPost).Execute(nil))
	require.NoError(t, c.PUT(t.Context(), "/").WithPath("/test//", http.MethodPut).Execute(nil))
	require.NoError(t, c.DELETE(t.Context(), "/").WithPath("/test//", http.MethodDelete).Execute(nil))
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
	require.NoError(t, c.GET(t.Context(), "test").Execute(nil))
}

func TestClient_WithHeader(t *testing.T) {
	t.Run("override accept header", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/octet-stream", r.Header.Get("Accept"))
		}))

		defer s.Close()

		c := NewClient(Config{BaseURL: must(url.Parse(s.URL))})
		err := c.GET(t.Context(), "/test").
			WithHeader("Accept", "application/octet-stream").
			Execute(nil)
		require.NoError(t, err)
	})

	t.Run("custom header", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			assert.Equal(t, "custom-value", r.Header.Get("X-Custom"))
		}))
		defer s.Close()

		c := NewClient(Config{BaseURL: must(url.Parse(s.URL))})
		err := c.GET(t.Context(), "/test").
			WithHeader("X-Custom", "custom-value").
			Execute(nil)
		require.NoError(t, err)
	})

	t.Run("empty string value", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t, r.Header.Get("X-Empty"))
		}))
		defer s.Close()

		c := NewClient(Config{BaseURL: must(url.Parse(s.URL))})
		err := c.GET(t.Context(), "/test").
			WithHeader("X-Empty", "").
			Execute(nil)
		require.NoError(t, err)
	})
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
		c := new(ClientImpl)
		assert.EqualError(t, c.GET(t.Context(), "/test").Execute(nil), "build URL: missing base URL")
	})

	t.Run("invalid json body", func(t *testing.T) {
		c := NewClient(Config{BaseURL: must(url.Parse("http://foo.bar/api")), HTTPClient: &http.Client{}})
		assert.Error(t, c.GET(t.Context(), "/test").WithJSONBody(brokenModel{}).Execute(nil))
	})
}

func TestClient_TokenTypes(t *testing.T) {
	var expectAuthHeader string

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectAuthHeader, r.Header.Get("Authorization"))
	}))
	defer s.Close()

	c := NewClient(Config{
		BaseURL:   must(url.Parse(s.URL)),
		APIToken:  "api",
		PaasToken: "paas",
	})

	t.Run("default", func(t *testing.T) {
		expectAuthHeader = "Api-Token api"
		assert.NoError(t, c.GET(t.Context(), "/test").Execute(nil))
	})

	t.Run("paas", func(t *testing.T) {
		expectAuthHeader = "Api-Token paas"
		assert.NoError(t, c.GET(t.Context(), "/test").WithPaasToken().Execute(nil))
	})

	t.Run("without token", func(t *testing.T) {
		expectAuthHeader = ""
		assert.NoError(t, c.GET(t.Context(), "/test").WithoutToken().Execute(nil))
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

func TestClient_ExecuteWriter(t *testing.T) {
	const responseBody = "binary-blob-content"
	const etagValue = `"v1"`

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fail":
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte(`{"error":{}}`))
		case "/not-modified":
			w.WriteHeader(http.StatusNotModified)
		default:
			w.Header().Set("ETag", etagValue)
			_, _ = w.Write([]byte(responseBody))
		}
	}))
	defer s.Close()

	c := NewClient(Config{BaseURL: must(url.Parse(s.URL))})

	t.Run("streams response body to writer and returns headers", func(t *testing.T) {
		var buf bytes.Buffer
		headers, err := c.GET(t.Context(), "/test").ExecuteWriter(&buf)
		require.NoError(t, err)
		assert.Equal(t, responseBody, buf.String())
		assert.Equal(t, etagValue, headers.Get("ETag"))
	})

	t.Run("returns error and writes nothing on non-2xx", func(t *testing.T) {
		var buf bytes.Buffer
		headers, err := c.GET(t.Context(), "/fail").ExecuteWriter(&buf)
		require.Error(t, err)
		assert.Nil(t, headers)
		assert.Empty(t, buf.String())
	})

	t.Run("returns HTTPError with status 304 on Not Modified", func(t *testing.T) {
		var buf bytes.Buffer
		headers, err := c.GET(t.Context(), "/not-modified").ExecuteWriter(&buf)
		require.Error(t, err)
		assert.True(t, HasStatusCode(err, http.StatusNotModified))
		assert.Nil(t, headers)
		assert.Empty(t, buf.String())
	})

	t.Run("returns error on missing base URL", func(t *testing.T) {
		var buf bytes.Buffer
		headers, err := new(ClientImpl).GET(t.Context(), "/test").ExecuteWriter(&buf)
		require.EqualError(t, err, "build URL: missing base URL")
		assert.Nil(t, headers)
		assert.Empty(t, buf.String())
	})

	t.Run("returns error on broken writer", func(t *testing.T) {
		headers, err := c.GET(t.Context(), "/test").ExecuteWriter(brokenWriter{})
		require.ErrorContains(t, err, "stream response body")
		assert.Nil(t, headers)
	})
}

type brokenWriter struct{}

func (brokenWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestClient_Execute_Cacheable(t *testing.T) {
	newCachingClient := func(t *testing.T, server *httptest.Server) *ClientImpl {
		t.Helper()
		transport := middleware.NewCacheRoundTripper(http.DefaultTransport, time.Minute)

		return NewClient(Config{
			BaseURL:    must(url.Parse(server.URL)),
			HTTPClient: &http.Client{Transport: transport},
		})
	}

	t.Run("non-empty response is cached", func(t *testing.T) {
		calls := 0
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			_, _ = w.Write([]byte(`{"foo":"bar"}`))
		}))
		defer s.Close()

		c := newCachingClient(t, s)

		var m1 cacheableModel
		require.NoError(t, c.GET(t.Context(), "/test").Execute(&m1))
		assert.Equal(t, 1, calls)
		assert.Equal(t, "bar", m1.Foo)

		var m2 cacheableModel
		require.NoError(t, c.GET(t.Context(), "/test").Execute(&m2))
		assert.Equal(t, 1, calls, "second call must hit cache, not backend")
		assert.Equal(t, "bar", m2.Foo)
	})

	t.Run("empty response invalidates cache", func(t *testing.T) {
		calls := 0
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			_, _ = w.Write([]byte(`{"foo":""}`))
		}))
		defer s.Close()

		c := newCachingClient(t, s)

		var m1 cacheableModel
		require.NoError(t, c.GET(t.Context(), "/test").Execute(&m1))
		assert.Equal(t, 1, calls)
		assert.True(t, m1.IsEmpty())

		var m2 cacheableModel
		require.NoError(t, c.GET(t.Context(), "/test").Execute(&m2))
		assert.Equal(t, 2, calls, "empty response must invalidate cache; second call must reach backend")
	})

	t.Run("non-Cacheable model is not cached", func(t *testing.T) {
		calls := 0
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			_, _ = w.Write([]byte(`{"foo":"bar"}`))
		}))
		defer s.Close()

		c := newCachingClient(t, s)

		var m1 apiModel // does not implement Cacheable
		require.NoError(t, c.GET(t.Context(), "/test").Execute(&m1))
		assert.Equal(t, 1, calls)

		var m2 apiModel
		require.NoError(t, c.GET(t.Context(), "/test").Execute(&m2))
		assert.Equal(t, 2, calls, "non-Cacheable model must not be cached")
	})

	t.Run("HTTP error invalidates cached entry", func(t *testing.T) {
		calls := 0
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{}}`))

				return
			}
			_, _ = w.Write([]byte(`{"foo":"recovered"}`))
		}))
		defer s.Close()

		c := newCachingClient(t, s)

		var m1 cacheableModel
		err := c.GET(t.Context(), "/test").Execute(&m1)
		require.Error(t, err)
		assert.Equal(t, 1, calls)

		// The cached 500 must have been invalidated by Execute; backend must be called again
		var m2 cacheableModel
		require.NoError(t, c.GET(t.Context(), "/test").Execute(&m2))
		assert.Equal(t, 2, calls, "HTTP error must invalidate cached entry")
		assert.Equal(t, "recovered", m2.Foo)
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
