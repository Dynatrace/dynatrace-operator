package core

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
)

func newTestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
	}
}

func TestHandleErrorResponse_SingleServerError(t *testing.T) {
	rb := &requestBuilder{}
	resp := newTestResponse(400, `{"error":{"code":400,"message":"bad request"}}`)
	err := rb.handleErrorResponse(resp, []byte(`{"error":{"code":400,"message":"bad request"}}`))
	httpErr := &HTTPError{}
	ok := errors.As(err, &httpErr)
	if !ok || httpErr.SingleError == nil || httpErr.SingleError.Code != 400 {
		t.Fatalf("Expected single server error, got %+v", err)
	}
	if got := httpErr.Error(); got != "HTTP 400: dynatrace server error 400: bad request" {
		t.Errorf("Unexpected error string: %s", got)
	}
}

func TestHandleErrorResponse_MultipleServerErrors(t *testing.T) {
	rb := &requestBuilder{}
	resp := newTestResponse(400, `[{"error":{"code":400,"message":"bad1"}},{"error":{"code":400,"message":"bad2"}}]`)
	err := rb.handleErrorResponse(resp, []byte(`[{"error":{"code":400,"message":"bad1"}},{"error":{"code":400,"message":"bad2"}}]`))
	httpErr := &HTTPError{}
	ok := errors.As(err, &httpErr)
	if !ok || len(httpErr.ServerErrors) != 2 {
		t.Fatalf("Expected multiple server errors, got %+v", err)
	}
	want := "HTTP 400: dynatrace server error 400: bad1; dynatrace server error 400: bad2"
	if got := httpErr.Error(); got != want {
		t.Errorf("Unexpected error string: %s", got)
	}
}

func TestHandleErrorResponse_GenericHTTPError(t *testing.T) {
	rb := &requestBuilder{
		path: "/test",
	}
	resp := newTestResponse(500, "not-json")
	err := rb.handleErrorResponse(resp, []byte("not-json"))
	httpErr := &HTTPError{}
	ok := errors.As(err, &httpErr)
	if !ok {
		t.Fatalf("Expected HTTPError, got %+v", err)
	}
	if httpErr.SingleError != nil || len(httpErr.ServerErrors) != 0 {
		t.Errorf("Expected no server errors, got %+v", httpErr)
	}
	want := "HTTP request (/test) failed 500"
	if got := httpErr.Error(); got != want {
		t.Errorf("Unexpected error string: %s", got)
	}
}

func TestHTTPError_ErrorMethod(t *testing.T) {
	httpErr := &HTTPError{
		StatusCode: 404,
		Message:    "not found",
	}
	if got := httpErr.Error(); got != "not found" {
		t.Errorf("Expected 'not found', got %s", got)
	}

	httpErr = &HTTPError{
		StatusCode: 401,
		SingleError: &ServerError{
			Code:    401,
			Message: "unauthorized",
		},
	}
	if got := httpErr.Error(); got != "HTTP 401: dynatrace server error 401: unauthorized" {
		t.Errorf("Unexpected error string: %s", got)
	}

	httpErr = &HTTPError{
		StatusCode: 400,
		ServerErrors: []ServerError{
			{Code: 400, Message: "bad1"},
			{Code: 400, Message: "bad2"},
		},
	}
	want := "HTTP 400: dynatrace server error 400: bad1; dynatrace server error 400: bad2"
	if got := httpErr.Error(); got != want {
		t.Errorf("Unexpected error string: %s", got)
	}
}
