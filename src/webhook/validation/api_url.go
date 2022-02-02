package validation

import (
	"net/url"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

const (
	exampleApiUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
	errorNoApiUrl = `The DynaKube's specification is missing the API URL or still has the example value set.
	Make sure you correctly specify the URL in your custom resource.
	`

	errorInvalidApiUrl = `The DynaKube's specification has an invalid API URL value set.
	Make sure you correctly specify the URL in your custom resource (including the /api postfix).
	`
)

func noApiUrl(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	apiUrl := dynakube.Spec.APIURL

	if apiUrl == exampleApiUrl {
		log.Info("api url is an example url", "apiUrl", apiUrl)
		return errorNoApiUrl
	}

	if apiUrl == "" {
		log.Info("requested dynakube has no api url", "name", dynakube.Name, "namespace", dynakube.Namespace)
		return errorNoApiUrl
	}

	return ""
}

func isInvalidApiUrl(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	apiUrl := dynakube.Spec.APIURL

	if !strings.HasSuffix(apiUrl, "/api") {
		log.Info("api url does not end with /api", "apiUrl", apiUrl)
		return errorInvalidApiUrl
	}

	parsedUrl, err := url.Parse(apiUrl)
	if err != nil {
		log.Info("API URL is not a valid URL", "err", err.Error())
		return errorInvalidApiUrl
	}

	fqdn := parsedUrl.Hostname()
	hostnameWithDomains := strings.FieldsFunc(fqdn,
		func(r rune) bool { return r == '.' },
	)

	if len(hostnameWithDomains) < 2 || len(hostnameWithDomains[0]) == 0 {
		log.Info("problem getting tenant id from fqdn", "fqdn", fqdn)
		return errorInvalidApiUrl
	}

	return ""
}
