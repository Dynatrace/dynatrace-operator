package validation

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

const (
	errorInvalidAPIServer = `The EdgeConnect's specification has an invalid apiServer value set.
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

func isInvalidAPIServer(_ context.Context, _ *Validator, ec *edgeconnect.EdgeConnect) string {
	for _, suffix := range allowedSuffix {
		if strings.HasSuffix(ec.Spec.APIServer, suffix) {
			hostnameWithDomains := strings.FieldsFunc(suffix,
				func(r rune) bool { return r == '.' },
			)

			hostnameWithTenant := strings.FieldsFunc(ec.Spec.APIServer,
				func(r rune) bool { return r == '.' },
			)

			if len(hostnameWithTenant) > len(hostnameWithDomains) {
				return ""
			}

			log.Info("apiServer is not a valid hostname", "apiServer", ec.Spec.APIServer)

			break
		}
	}

	return errorInvalidAPIServer
}
