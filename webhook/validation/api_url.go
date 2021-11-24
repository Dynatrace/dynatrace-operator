package validation

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
)

const (
	exampleApiUrl = "https://ENVIRONMENTID.live.dynatrace.com/api"
	errorNoApiUrl = `The DynaKube's specification is missing the API URL or still has the example value set.
	Make sure you correctly specify the URL in your custom resource.
	`
)

func noApiUrl(dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if dynakube.Spec.APIURL != "" && dynakube.Spec.APIURL != exampleApiUrl {
		return ""
	}
	log.Info("requested dynakube has no api url", "name", dynakube.Name, "namespace", dynakube.Namespace)
	return errorNoApiUrl
}
