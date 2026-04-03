package core

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

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
		{"", "", levelDisabled},
		{"info", "default", levelDisabled},
		{"info", "request", levelDisabled},
		{"info", "response", levelDisabled},
		{"info", "full", levelDisabled},

		{"debug", "", levelDefault},
		{"debug", "default", levelDefault},
		{"debug", "request", levelRequest},
		{"debug", "response", levelResponse},
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
	bearerToken := "Bearer someOpaqueToken"

	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"response-foo": []string{"response-" + token}},
		Request: &http.Request{
			Method: http.MethodGet,
			URL:    u,
			Header: http.Header{
				"request-foo":   []string{"request-" + token},
				"authorization": []string{bearerToken},
			},
		},
	}

	requestBody := []byte("request " + token + "rest")
	responseBody := []byte("response " + token + "rest")

	tests := []struct {
		name     string
		logLevel int
		want     []any
	}{
		{"disabled", levelDisabled, nil},
		{
			"default",
			levelDefault,
			[]any{
				"method", "GET", "host", "host.test", "path", "/path-foo", "query", `{"query-foo":"query-bar"}`, "status_code", 200, "duration", "1s",
			},
		},
		{
			"request",
			levelRequest,
			[]any{
				"method", "GET", "host", "host.test", "path", "/path-foo", "query", `{"query-foo":"query-bar"}`, "status_code", 200, "duration", "1s",
				"request_body", "request " + publicPart + ".***rest",
			},
		},
		{
			"response",
			levelResponse,
			[]any{
				"method", "GET", "host", "host.test", "path", "/path-foo", "query", `{"query-foo":"query-bar"}`, "status_code", 200, "duration", "1s",
				"request_body", "request " + publicPart + ".***rest",
				"response_body", "response " + publicPart + ".***rest",
			},
		},
		{
			"full",
			levelFull,
			[]any{
				"method", "GET", "host", "host.test", "path", "/path-foo", "query", `{"query-foo":"query-bar"}`, "status_code", 200, "duration", "1s",
				"request_headers", `{"Authorization":"Bearer ***","Request-Foo":"request-` + publicPart + `.***"}`,
				"response_headers", `{"Response-Foo":"response-` + publicPart + `.***"}`,
				"request_body", "request " + publicPart + ".***rest",
				"response_body", "response " + publicPart + ".***rest",
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

			get := createLoggerArgs(requestBody)
			// Advance time to always get duration=1s
			fixedTime = fixedTime.Add(1 * time.Second)

			assert.Equal(t, tt.want, get(response, responseBody))
		})
	}
}

func Test_dumpValues(t *testing.T) {
	token := strings.Repeat("a", 5) + "." + strings.Repeat("A", 8) + "." + strings.Repeat("B", 64)
	authorizationToken := "test-token"

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

		{"mask secret w canonicalize", http.Header{"authorization": []string{"Bearer " + authorizationToken}, "request-foo": []string{"request-" + token}}, true, `{"Authorization":"Bearer ***","Request-Foo":"request-aaaaa.AAAAAAAA.***"}`},
		{"mask secret wo canonicalize", url.Values{"authorization": []string{"Bearer " + authorizationToken}, "request-foo": []string{"request-" + token}}, false, `{"authorization":"Bearer ***","request-foo":"request-aaaaa.AAAAAAAA.***"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dumpValues(tt.header, tt.canonicalize)
			assert.Equal(t, tt.want, got)
		})
	}
}
