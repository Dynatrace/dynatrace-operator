package edgeconnect

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
)

const (
	errorInvalidApiServer = `The EdgeConnect's specification has an invalid apiServer value set.
	Make sure you correctly specify the apiServer in your custom resource.
	`
)

var (
	allowedSuffix = []string{
		".dev.apps.dynatracelabs.com",
		".sprint.apps.dynatracelabs.com",
		".apps.dynatrace.com",
	}
)

func IsInvalidApiServer(_ *edgeconnectValidator, edgeConnect *edgeconnect.EdgeConnect) string {
	for _, suffix := range allowedSuffix {
		if strings.HasSuffix(edgeConnect.Spec.ApiServer, suffix) {
			hostnameWithDomains := strings.FieldsFunc(suffix,
				func(r rune) bool { return r == '.' },
			)

			hostnameWithTenant := strings.FieldsFunc(edgeConnect.Spec.ApiServer,
				func(r rune) bool { return r == '.' },
			)

			if len(hostnameWithTenant) > len(hostnameWithDomains) {
				return ""
			}
			log.Info("apiServer is not a valid hostname", "apiServer", edgeConnect.Spec.ApiServer)
			break
		}
	}
	return errorInvalidApiServer
}
