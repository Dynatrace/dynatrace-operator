package validation

import (
	"context"
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

const (
	errorInvalidAPIServer = `The EdgeConnect's specification has an invalid apiServer value set.
	Make sure you correctly specify the apiServer in your custom resource.
	`

	errorMissingAllowedSuffixAPIServer = `The EdgeConnect's specification has an invalid apiServer value set.
	Example valid values:
	-  For prod: "<tenantID>.apps.dynatrace.com"
	-  For dev: "<tenantID>.dev.apps.dynatracelabs.com"
	-  For sprint: "<tenantID>.sprint.apps.dynatracelabs.com"
	Make sure you correctly specify the apiServer in your custom resource.
	`

	errorProtocolIsNotAllowedAPIServer = `The EdgeConnect's specification has an invalid apiServer value set.
	Should NOT include protocol.
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

func isAllowedSuffixAPIServer(_ context.Context, _ *validatorClient, ec *edgeconnect.EdgeConnect) string {
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

	return errorMissingAllowedSuffixAPIServer
}

func checkAPIServerProtocolNotSet(_ context.Context, _ *validatorClient, ec *edgeconnect.EdgeConnect) string {
	parsedURL, err := url.Parse(ec.Spec.APIServer)
	if err != nil {
		log.Info("API Server URL is not a valid URL", "err", err.Error())

		return errorInvalidAPIServer
	}

	if parsedURL.Scheme != "" {
		return errorProtocolIsNotAllowedAPIServer
	}

	return ""
}
