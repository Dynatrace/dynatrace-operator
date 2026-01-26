package validation

import (
	"context"
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	ExampleAPIURL = "https://ENVIRONMENTID.live.dynatrace.com/api"
	errorNoAPIURL = `The DynaKube's specification is missing the API URL or still has the example value set.
	Make sure you correctly specify the URL in your custom resource.
	`

	errorInvalidAPIURL = `The DynaKube's specification has an invalid API URL value set.
	Make sure you correctly specify the URL in your custom resource (including the /api postfix).
	`

	errorThirdGenAPIURL = `The DynaKube's specification has an 3rd gen API URL. Make sure to remove the 'apps' part
	out of it. Example: ` + ExampleAPIURL

	errorMutatedAPIURL = `The DynaKube's specification mutated the API URL although it is immutable. Please delete the CR and then apply a new one`
)

func NoAPIURL(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
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

func IsInvalidAPIURL(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	apiURL := dk.Spec.APIURL

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

func IsThirdGenAPIUrl(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	if strings.Contains(dk.APIURL(), ".apps.") {
		return errorThirdGenAPIURL
	}

	return ""
}

func IsMutatedAPIURL(_ context.Context, _ *validatorClient, oldDk *dynakube.DynaKube, newDk *dynakube.DynaKube) string {
	if oldDk.Spec.APIURL != newDk.Spec.APIURL {
		return errorMutatedAPIURL
	}

	return ""
}
