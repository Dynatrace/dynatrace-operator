package core

import (
	"encoding/json"
	"net/http"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	levelDisabled = iota
	levelDefault  // default
	levelRequest  // request
	levelResponse // response
	levelFull     // full
)

const (
	// LogLevelEnv controls the verbosity of the Dynatrace API client.
	// The value will only be used when the LOG_LEVEL variable is set to "debug".
	LogLevelEnv = "DT_CLIENT_LOG_LEVEL"

	// CacheHitHeader is set on responses served from the in-memory cache so that
	// the core client can include a "cached" field in its log output.
	CacheHitHeader = "X-DT-Cache"

	// CacheSkipHeader can be set on a request to bypass the in-memory cache for
	// that specific request. Any non-empty value disables both cache reads and writes.
	CacheSkipHeader = "X-DT-Cache-Skip"
)

var logLevel = getLogLevel()

func getLogLevel() int {
	if strings.ToLower(os.Getenv(logd.LogLevelEnv)) == "debug" {
		switch strings.ToLower(os.Getenv(LogLevelEnv)) {
		case "full":
			return levelFull
		case "request":
			return levelRequest
		case "response":
			return levelResponse
		default:
			return levelDefault
		}
	}

	return levelDisabled
}

// for unit tests
var timeNow = time.Now

// createLoggerArgs should be called before making an HTTP request.
// Calling the returned closure will yield key/value pairs that can be used for a logr.Logger.
// The key/value pairs are generated according to the inputs and the configured log level.
func createLoggerArgs(requestBody []byte) func(resp *http.Response, responseBody []byte) []any {
	start := timeNow()

	return func(resp *http.Response, responseBody []byte) []any {
		if logLevel < levelDefault {
			return nil
		}

		duration := timeNow().Sub(start)

		args := []any{
			"method", resp.Request.Method,
			"host", resp.Request.URL.Host,
			"path", resp.Request.URL.Path,
			"query", dumpValues(resp.Request.URL.Query(), false),
			"status_code", resp.StatusCode,
			"duration", duration.String(),
			"cached", resp.Header.Get(CacheHitHeader) != "",
		}

		if logLevel >= levelFull {
			args = append(args, "request_headers", dumpValues(resp.Request.Header, true))
			args = append(args, "response_headers", dumpValues(resp.Header, true))
		}

		if logLevel >= levelRequest {
			args = append(args, "request_body", sanitizeBody(requestBody))
		}

		if logLevel >= levelResponse {
			args = append(args, "response_body", sanitizeBody(responseBody))
		}

		return args
	}
}

// Detect Dynatrace tokens in the format of <prefix>.<public>.<private>:
//   - Prefix is expected to have at least 5 alphanum characters.
//   - Public part can either have 8 or 24 base32 characters.
//   - Private part is 64 base32 characters.
var dtTokenRegex = regexp.MustCompile(`[a-z0-9]{5,}\.([A-Z0-7]{8}|[A-Z0-7]{24})\.[A-Z0-7]{64}`)

// Detect JWT tokens
var jwtBearerTokenRegex = regexp.MustCompile(`ey[A-Za-z0-9-_]+\.ey[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+`)

func sanitizeBody(body []byte) string {
	// Only hide private parts from output
	sanitized := dtTokenRegex.ReplaceAllStringFunc(string(body), func(s string) string {
		idx := strings.LastIndexByte(s, '.')

		return s[:idx] + ".***"
	})

	sanitized = jwtBearerTokenRegex.ReplaceAllStringFunc(sanitized, func(s string) string {
		idx := strings.LastIndexByte(s, '.')

		return s[:idx] + ".***"
	})

	return sanitized
}

// Dump objects like http.Header or url.Values into a JSON string.
// The boolean controls whether the keys will be canonicalized into MIME header format.
func dumpValues(header map[string][]string, canonicalize bool) string {
	if len(header) == 0 {
		// empty value should lead to empty string (not '{}')
		return ""
	}

	data := make(map[string]any, len(header))

	for key, values := range header {
		if canonicalize {
			key = textproto.CanonicalMIMEHeaderKey(key)
		}

		switch len(values) {
		case 0:
			data[key] = ""
		case 1:
			data[key] = values[0]
		default:
			data[key] = values
		}
	}

	o, _ := json.Marshal(data)

	return sanitizeBody(o)
}
