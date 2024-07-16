package validation

import (
	"context"
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	ExampleApiUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
	errorNoApiUrl = `The DynaKube's specification is missing the API URL or still has the example value set.
	Make sure you correctly specify the URL in your custom resource.
	`

	errorInvalidApiUrl = `The DynaKube's specification has an invalid API URL value set.
	Make sure you correctly specify the URL in your custom resource (including the /api postfix).
	`

	errorThirdGenApiUrl = `The DynaKube's specification has an 3rd gen API URL. Make sure to remove the 'apps' part
	out of it. Example: ` + ExampleApiUrl
)

func NoApiUrl(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	apiUrl := dk.Spec.APIURL

	if apiUrl == ExampleApiUrl {
		log.Info("api url is an example url", "apiUrl", apiUrl)

		return errorNoApiUrl
	}

	if apiUrl == "" {
		log.Info("requested dynakube has no api url", "name", dk.Name, "namespace", dk.Namespace)

		return errorNoApiUrl
	}

	return ""
}

func IsInvalidApiUrl(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	apiUrl := dk.Spec.APIURL

	if !strings.HasSuffix(apiUrl, "/api") {
		log.Info("api url does not end with /api", "apiUrl", apiUrl)

		return errorInvalidApiUrl
	}

	parsedUrl, err := url.Parse(apiUrl)
	if err != nil {
		log.Info("API URL is not a valid URL", "err", err.Error())

		return errorInvalidApiUrl
	}

	hostname := parsedUrl.Hostname()
	hostnameWithDomains := strings.FieldsFunc(hostname,
		func(r rune) bool { return r == '.' },
	)

	if len(hostnameWithDomains) < 1 || len(hostnameWithDomains[0]) == 0 {
		log.Info("invalid hostname in the api url", "hostname", hostname)

		return errorInvalidApiUrl
	}

	return ""
}

func IsThirdGenAPIUrl(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if strings.Contains(dk.ApiUrl(), ".apps.") {
		return errorThirdGenApiUrl
	}

	return ""
}
