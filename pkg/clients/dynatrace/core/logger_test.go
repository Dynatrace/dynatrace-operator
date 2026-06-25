package core

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getLogLevel(t *testing.T) {
	tests := []struct {
		logLevelEnv    string
		clientDebugEnv string
		want           int
	}{
		{"", "", levelDefault},
		{"info", "", levelDefault},
		{"", "disabled", levelDisabled},
		{"", "request", levelRequest},
		{"", "response", levelResponse},
		{"", "full", levelFull},
		{"debug", "", levelFull},
		{"debug", "disabled", levelFull},
		{"debug", "request", levelFull},
		{"debug", "full", levelFull},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("log=%s debug=%s", tt.logLevelEnv, tt.clientDebugEnv), func(t *testing.T) {
			t.Setenv(logd.LogLevelEnv, tt.logLevelEnv)
			t.Setenv(LogLevelEnv, tt.clientDebugEnv)
			assert.Equal(t, tt.want, getLogLevel())
		})
	}
}

func Test_loggerArgs(t *testing.T) {
	u, err := url.Parse("https://host.test/path-foo?query-foo=query-bar")
	require.NoError(t, err)

	publicPart := strings.Repeat("a", 5) + "." + strings.Repeat("B", 24)
	token := publicPart + "." + strings.Repeat("C", 64)
	jwtBearerToken := "eya.eyb.c"

	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"response-foo": []string{"response-" + token}},
		Request: &http.Request{
			Method: http.MethodGet,
			URL:    u,
			Header: http.Header{
				"request-foo":   []string{"request-" + token},
				"authorization": []string{"Bearer " + jwtBearerToken},
			},
		},
	}

	requestBody := []byte("request " + "Bearer " + jwtBearerToken + " " + token + "rest")
	responseBody := []byte("response " + "Bearer " + jwtBearerToken + " " + token + "rest")

	sanitizedRequest := "request " + "Bearer " + "eya.eyb.***" + " " + publicPart + ".***rest"
	sanitizedResponse := "response " + "Bearer " + "eya.eyb.***" + " " + publicPart + ".***rest"

	tests := []struct {
		name     string
		logLevel int
		want     []any
	}{
		{"disabled", levelDisabled, nil},
		{"default", levelDefault, nil},
		{"default/cached", levelDefault, nil},
		{
			"default/error",
			levelDefault,
			[]any{
				"method", "GET", "path", "/path-foo", "status_code", http.StatusBadRequest, "duration", "1s",
			},
		},
		{
			"request",
			levelRequest,
			[]any{
				"method", "GET", "path", "/path-foo", "status_code", 200, "duration", "1s",
				"request_body", sanitizedRequest,
			},
		},
		{
			"response",
			levelResponse,
			[]any{
				"method", "GET", "path", "/path-foo", "status_code", 200, "duration", "1s",
				"request_body", sanitizedRequest,
				"response_body", sanitizedResponse,
			},
		},
		{
			"full",
			levelFull,
			[]any{
				"method", "GET", "path", "/path-foo", "status_code", 200, "duration", "1s",
				"cached", false,
				"host", "host.test",
				"query", `{"query-foo":"query-bar"}`,
				"request_headers", `{"Authorization":"Bearer eya.eyb.***","Request-Foo":"request-` + publicPart + `.***"}`,
				"response_headers", `{"Response-Foo":"response-` + publicPart + `.***"}`,
				"request_body", sanitizedRequest,
				"response_body", sanitizedResponse,
			},
		},
		{
			"full/cached",
			levelFull,
			[]any{
				"method", "GET", "path", "/path-foo", "status_code", 200, "duration", "1s",
				"cached", true,
				"host", "host.test",
				"query", `{"query-foo":"query-bar"}`,
				"request_headers", `{"Authorization":"Bearer eya.eyb.***","Request-Foo":"request-` + publicPart + `.***"}`,
				"response_headers", `{"Response-Foo":"response-` + publicPart + `.***","X-Dt-Cache":"true"}`,
				"request_body", sanitizedRequest,
				"response_body", sanitizedResponse,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVerbosity := logLevel
			logLevel = tt.logLevel

			fixedTime := time.Now().Round(time.Second)
			timeNow = func() time.Time { return fixedTime }

			t.Cleanup(func() {
				logLevel = oldVerbosity
				timeNow = time.Now
			})

			resp := response
			switch tt.name {
			case "default/error":
				errResp := *response
				errResp.StatusCode = http.StatusBadRequest
				resp = &errResp
			case "default/cached", "full/cached":
				cached := *response
				cached.Header = response.Header.Clone()
				cached.Header.Set(middleware.CacheHitHeader, "true")
				resp = &cached
			}

			get := createLoggerArgs(requestBody)
			// Advance time to always get duration=1s
			fixedTime = fixedTime.Add(1 * time.Second)

			assert.Equal(t, tt.want, get(resp, responseBody))
		})
	}
}

func Test_dumpValues(t *testing.T) {
	token := strings.Repeat("a", 5) + "." + strings.Repeat("A", 8) + "." + strings.Repeat("B", 64)
	authorizationToken := "eya.eyb.c"

	tests := []struct {
		name         string
		header       map[string][]string
		canonicalize bool
		want         string
	}{
		{"empty w canonicalize", nil, true, ""},
		{"empty wo canonicalize", nil, false, ""},

		{"empty value w canonicalize", http.Header{"x-foo": nil}, true, `{"X-Foo":""}`},
		{"empty value wo canonicalize", url.Values{"foo": nil}, false, `{"foo":""}`},

		{"single value w canonicalize", http.Header{"x-foo": []string{"bar"}}, true, `{"X-Foo":"bar"}`},
		{"single value wo canonicalize", url.Values{"foo": []string{"bar"}}, false, `{"foo":"bar"}`},

		{"multi value w canonicalize", http.Header{"x-foo": []string{"bar", "baz"}}, true, `{"X-Foo":["bar","baz"]}`},
		{"multi value wo canonicalize", url.Values{"foo": []string{"bar", "baz"}}, false, `{"foo":["bar","baz"]}`},

		{"mask secret w canonicalize", http.Header{"authorization": []string{"Bearer " + authorizationToken}, "request-foo": []string{"request-" + token}}, true, `{"Authorization":"Bearer eya.eyb.***","Request-Foo":"request-aaaaa.AAAAAAAA.***"}`},
		{"mask secret wo canonicalize", url.Values{"authorization": []string{"Bearer " + authorizationToken}, "request-foo": []string{"request-" + token}}, false, `{"authorization":"Bearer eya.eyb.***","request-foo":"request-aaaaa.AAAAAAAA.***"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dumpValues(tt.header, tt.canonicalize)
			assert.Equal(t, tt.want, got)
		})
	}
}
