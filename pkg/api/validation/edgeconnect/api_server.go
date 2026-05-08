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

func isAllowedSuffixAPIServer(_ context.Context, _ *Validator, ec *edgeconnect.EdgeConnect) string {
	for _, suffix := range allowedSuffix {
		if strings.HasSuffix(ec.Spec.APIServer, suffix) && len(ec.Spec.APIServer) > len(suffix) {
			return ""
		}
	}

	return errorMissingAllowedSuffixAPIServer
}

func checkAPIServerProtocolNotSet(_ context.Context, _ *Validator, ec *edgeconnect.EdgeConnect) string {
	parsedURL, err := url.Parse(ec.Spec.APIServer)
	if err != nil {
		return errorInvalidAPIServer
	}

	if parsedURL.Scheme != "" {
		return errorProtocolIsNotAllowedAPIServer
	}

	return ""
}
