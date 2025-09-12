package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func newTestConfig(base string) CoreClient {
	u, _ := url.Parse(base)

	return CoreClient{
		BaseURL:         u,
		HTTPClient:      http.DefaultClient,
		UserAgent:       "test-agent",
		APIToken:        "api-token",
		PaasToken:       "paas-token",
		DataIngestToken: "data-token",
	}
}

func TestRequestBuilder_BuilderMethods(t *testing.T) {
	config := newTestConfig("http://localhost").newRequest(t.Context())
	rb := config.GET(t.Context(), "/test").(*CoreClient)

	method := "POST"
	if rb.withMethod(method).(*CoreClient).method != method {
		t.Errorf("WithMethod failed")
	}

	path := "/test"
	if rb.WithPath(path).(*CoreClient).path != path {
		t.Errorf("WithPath failed")
	}

	key, value := "foo", "bar"
	if rb.WithQueryParam(key, value).(*CoreClient).queryParams[key] != value {
		t.Errorf("WithQueryParam failed")
	}

	params := map[string]string{"a": "1", "b": "2"}
	rb.WithQueryParams(params)
	for k, v := range params {
		if rb.queryParams[k] != v {
			t.Errorf("WithQueryParams failed for %s", k)
		}
	}

	body := map[string]string{"x": "y"}
	rb.WithJSONBody(body)
	var out map[string]string
	json.Unmarshal(rb.body, &out)
	if out["x"] != "y" {
		t.Errorf("WithJSONBody failed")
	}

	raw := []byte("raw-body")
	rb.WithRawBody(raw)
	if !bytes.Equal(rb.body, raw) {
		t.Errorf("WithRawBody failed")
	}

	rb.WithTokenType(TokenTypePaaS)
	if rb.tokenType != TokenTypePaaS {
		t.Errorf("WithTokenType failed")
	}

	rb.WithPaasToken()
	if rb.tokenType != TokenTypePaaS {
		t.Errorf("WithPaasToken failed")
	}
}

func TestRequestBuilder_Execute_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"foo":"bar"}`))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	config := CoreClient{
		BaseURL:    u,
		HTTPClient: server.Client(),
		UserAgent:  "test-agent",
		APIToken:   "api-token",
	}

	target := struct{ Foo string }{}
	err := config.GET(t.Context(), "").Execute(&target)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if target.Foo != "bar" {
		t.Errorf("Execute did not unmarshal response")
	}
}

func TestRequestBuilder_ExecuteRaw_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"foo":"bar"}`))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	config := CoreClient{
		BaseURL:    u,
		HTTPClient: server.Client(),
		UserAgent:  "test-agent",
		APIToken:   "api-token",
	}

	body, err := config.GET(t.Context(), "").ExecuteRaw()
	if err != nil {
		t.Fatalf("ExecuteRaw failed: %v", err)
	}
	if !bytes.Contains(body, []byte("foo")) {
		t.Errorf("ExecuteRaw did not return expected body")
	}
}

func TestRequestBuilder_Execute_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":400,"message":"bad request"}}`))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	config := CoreClient{
		BaseURL:    u,
		HTTPClient: server.Client(),
		UserAgent:  "test-agent",
		APIToken:   "api-token",
	}

	target := struct{}{}
	err := config.GET(t.Context(), "").Execute(&target)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	if !errors.Is(err, err) {
		t.Errorf("Expected error, got %v", err)
	}
}

func TestRequestBuilder_ExecuteRaw_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"code":500,"message":"server error"}}`))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	config := CoreClient{
		BaseURL:    u,
		HTTPClient: server.Client(),
		UserAgent:  "test-agent",
		APIToken:   "api-token",
	}

	_, err := config.GET(t.Context(), "").ExecuteRaw()
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestRequestBuilder_setHeaders(t *testing.T) {
	config := newTestConfig("http://localhost")
	rb := config.POST(t.Context(), "").(*CoreClient)
	req, _ := http.NewRequest(http.MethodPost, "http://localhost", nil)
	rb.setHeaders(req)
	if req.Header.Get("Authorization") == "" {
		t.Errorf("setHeaders did not set Authorization")
	}
	if req.Header.Get("User-Agent") != config.UserAgent {
		t.Errorf("setHeaders did not set User-Agent")
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("setHeaders did not set Content-Type for POST")
	}
}

func TestRequestBuilder_handleResponse_UnmarshalError(t *testing.T) {
	config := newTestConfig("http://localhost")
	rb := config.GET(t.Context(), "").(*CoreClient)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte("not-json"))),
	}
	target := struct{ Foo string }{}
	err := rb.handleResponse(resp, &target)
	if err == nil {
		t.Errorf("Expected unmarshal error, got nil")
	}
}

func TestCoreClient_GET_POST_PUT_DELETE(t *testing.T) {
	config := newTestConfig("http://localhost")
	if reflect.TypeOf(config.GET(t.Context(), "/foo")).String() != "*core.CoreClient" {
		t.Errorf("GET did not return *core.CoreClient, got: %s", reflect.TypeOf(config.GET(t.Context(), "/foo")).String())
	}
	if reflect.TypeOf(config.POST(t.Context(), "/foo")).String() != "*core.CoreClient" {
		t.Errorf("POST did not return *core.CoreClient, got: %s", reflect.TypeOf(config.POST(t.Context(), "/foo")).String())
	}
	if reflect.TypeOf(config.PUT(t.Context(), "/foo")).String() != "*core.CoreClient" {
		t.Errorf("PUT did not return *core.CoreClient, got: %s", reflect.TypeOf(config.PUT(t.Context(), "/foo")).String())
	}
	if reflect.TypeOf(config.DELETE(t.Context(), "/foo")).String() != "*core.CoreClient" {
		t.Errorf("DELETE did not return *core.CoreClient, got: %s", reflect.TypeOf(config.DELETE(t.Context(), "/foo")).String())
	}
}
