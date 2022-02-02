package validation

import (
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

const (
	exampleApiUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
	errorNoApiUrl = `The DynaKube's specification is missing the API URL or still has the example value set.
	Make sure you correctly specify the URL in your custom resource.
	`
	noError = ""
)

func noApiUrl(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	apiUrl := dynakube.Spec.APIURL

	if !strings.HasSuffix(apiUrl, "/api") {
		log.Info("api url does not end with /api", "apiUrl", apiUrl)
		return errorNoApiUrl
	}

	if apiUrl == exampleApiUrl {
		log.Info("api url is an example url", "apiUrl", apiUrl)
		return errorNoApiUrl
	}

	if apiUrl != "" {
		return noError
	}

	log.Info("requested dynakube has no api url", "name", dynakube.Name, "namespace", dynakube.Namespace)
	return errorNoApiUrl
}
