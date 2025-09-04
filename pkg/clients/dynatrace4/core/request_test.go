package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

type testTokenType string

func newTestConfig(base string) CommonConfig {
	u, _ := url.Parse(base)
	return CommonConfig{
		BaseURL:         u,
		HTTPClient:      http.DefaultClient,
		UserAgent:       "test-agent",
		APIToken:        "api-token",
		PaasToken:       "paas-token",
		DataIngestToken: "data-token",
	}
}

func TestRequestBuilder_BuilderMethods(t *testing.T) {
	config := newTestConfig("http://localhost")
	rb := NewRequest(config).(*requestBuilder)
	ctx := context.Background()

	rb2 := rb.WithContext(ctx).(*requestBuilder)
	if rb2.ctx != ctx {
		t.Errorf("WithContext failed")
	}

	method := "POST"
	if rb2.WithMethod(method).(*requestBuilder).method != method {
		t.Errorf("WithMethod failed")
	}

	path := "/test"
	if rb2.WithPath(path).(*requestBuilder).path != path {
		t.Errorf("WithPath failed")
	}

	key, value := "foo", "bar"
	if rb2.WithQueryParam(key, value).(*requestBuilder).queryParams[key] != value {
		t.Errorf("WithQueryParam failed")
	}

	params := map[string]string{"a": "1", "b": "2"}
	rb2.WithQueryParams(params)
	for k, v := range params {
		if rb2.queryParams[k] != v {
			t.Errorf("WithQueryParams failed for %s", k)
		}
	}

	body := map[string]string{"x": "y"}
	rb2.WithJSONBody(body)
	var out map[string]string
	json.Unmarshal(rb2.body, &out)
	if out["x"] != "y" {
		t.Errorf("WithJSONBody failed")
	}

	raw := []byte("raw-body")
	rb2.WithRawBody(raw)
	if !bytes.Equal(rb2.body, raw) {
		t.Errorf("WithRawBody failed")
	}

	rb2.WithTokenType(TokenTypePaaS)
	if rb2.tokenType != TokenTypePaaS {
		t.Errorf("WithTokenType failed")
	}

	rb2.WithPaasToken()
	if rb2.tokenType != TokenTypePaaS {
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
	config := CommonConfig{
		BaseURL:    u,
		HTTPClient: server.Client(),
		UserAgent:  "test-agent",
		APIToken:   "api-token",
	}

	target := struct{ Foo string }{}
	err := NewRequest(config).
		WithContext(context.Background()).
		WithMethod("GET").
		WithPath("").
		Execute(&target)
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
	config := CommonConfig{
		BaseURL:    u,
		HTTPClient: server.Client(),
		UserAgent:  "test-agent",
		APIToken:   "api-token",
	}

	body, err := NewRequest(config).
		WithContext(context.Background()).
		WithMethod("GET").
		WithPath("").
		ExecuteRaw()
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
	config := CommonConfig{
		BaseURL:    u,
		HTTPClient: server.Client(),
		UserAgent:  "test-agent",
		APIToken:   "api-token",
	}

	target := struct{}{}
	err := NewRequest(config).WithMethod("GET").WithPath("").Execute(&target)
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
	config := CommonConfig{
		BaseURL:    u,
		HTTPClient: server.Client(),
		UserAgent:  "test-agent",
		APIToken:   "api-token",
	}

	_, err := NewRequest(config).WithMethod("GET").WithPath("").ExecuteRaw()
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestRequestBuilder_setHeaders(t *testing.T) {
	config := newTestConfig("http://localhost")
	rb := NewRequest(config).(*requestBuilder)
	rb.method = http.MethodPost
	req, _ := http.NewRequest("POST", "http://localhost", nil)
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
	rb := NewRequest(config).(*requestBuilder)
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte("not-json"))),
	}
	target := struct{ Foo string }{}
	err := rb.handleResponse(resp, &target)
	if err == nil {
		t.Errorf("Expected unmarshal error, got nil")
	}
}

func TestCommonConfig_GET_POST_PUT_DELETE(t *testing.T) {
	config := newTestConfig("http://localhost")
	if reflect.TypeOf(config.GET("/foo")).String() != "*core.requestBuilder" {
		t.Errorf("GET did not return requestBuilder")
	}
	if reflect.TypeOf(config.POST("/foo")).String() != "*core.requestBuilder" {
		t.Errorf("POST did not return requestBuilder")
	}
	if reflect.TypeOf(config.PUT("/foo")).String() != "*core.requestBuilder" {
		t.Errorf("PUT did not return requestBuilder")
	}
	if reflect.TypeOf(config.DELETE("/foo")).String() != "*core.requestBuilder" {
		t.Errorf("DELETE did not return requestBuilder")
	}
}
