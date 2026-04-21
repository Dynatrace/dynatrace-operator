package validation

import (
	"context"
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
)

const (
	ExampleAPIURL = "https://ENVIRONMENTID.live.dynatrace.com/api"
	errorNoAPIURL = `The DynaKube's specification is missing the API URL or still has the example value set.
	Make sure you correctly specify the URL in your custom resource.
	`

	errorInvalidAPIURL = `The DynaKube's specification has an invalid API URL value set.
	Make sure you correctly specify the URL in your custom resource (including the /api postfix).
	`

	errorThirdGenAPIURL = `The DynaKube's specification has an 3rd gen API URL and the apiToken provided is not a platform token. Make sure to remove the 'apps' part
	out of it. Example: ` + ExampleAPIURL

	errorCheckingSecret = `Failed to check the DynaKube's secret to check for 3rd gen API URL. Make sure the secret exists and is accessible by the operator.`

	errorMutatedAPIURL = `The DynaKube's specification mutated the tenant in the API URL although it is immutable. Please delete the CR and then apply a new one`
)

func NoAPIURL(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	apiURL := dk.Spec.APIURL

	if apiURL == ExampleAPIURL {
		log.Info("api url is an example url", "apiUrl", apiURL)

		return errorNoAPIURL
	}

	if apiURL == "" {
		log.Info("requested dynakube has no api url", "name", dk.Name, "namespace", dk.Namespace)

		return errorNoAPIURL
	}

	return ""
}

func IsInvalidAPIURL(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	apiURL := dk.Spec.APIURL

	if isThirdGenAPIURL(apiURL) {
		return ""
	}

	if !strings.HasSuffix(apiURL, "/api") {
		log.Info("api url does not end with /api", "apiUrl", apiURL)

		return errorInvalidAPIURL
	}

	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		log.Info("API URL is not a valid URL", "err", err.Error())

		return errorInvalidAPIURL
	}

	hostname := parsedURL.Hostname()
	hostnameWithDomains := strings.FieldsFunc(hostname,
		func(r rune) bool { return r == '.' },
	)

	if len(hostnameWithDomains) < 1 || len(hostnameWithDomains[0]) == 0 {
		log.Info("invalid hostname in the api url", "hostname", hostname)

		return errorInvalidAPIURL
	}

	return ""
}

func isThirdGenAPIURL(apiURL string) bool {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return false
	}

	return strings.Contains(parsed.Hostname(), ".apps.")
}

func IsThirdGenAPIUrl(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if !isThirdGenAPIURL(dk.APIURL()) {
		return ""
	}

	tokenReader := token.NewReader(dv.apiReader, dk)

	hasPlatformToken, err := tokenReader.HasPlatformToken(ctx)
	if err != nil {
		return errorCheckingSecret
	}

	if !hasPlatformToken {
		return errorThirdGenAPIURL
	}

	return ""
}

func tenantUUIDFromAPIURL(apiURL string) string {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return apiURL
	}

	hostname := parsed.Hostname()
	if idx := strings.IndexByte(hostname, '.'); idx > 0 {
		return hostname[:idx]
	}

	return hostname
}

func IsMutatedAPIURL(_ context.Context, _ *Validator, oldDK *dynakube.DynaKube, newDK *dynakube.DynaKube) string {
	if tenantUUIDFromAPIURL(oldDK.Spec.APIURL) != tenantUUIDFromAPIURL(newDK.Spec.APIURL) {
		return errorMutatedAPIURL
	}

	return ""
}
