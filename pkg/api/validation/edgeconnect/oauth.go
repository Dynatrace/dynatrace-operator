package validation

import (
	"context"
	"net/url"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

const (
	errorInvalidOauthEndpoint = `The EdgeConnect's specification has an invalid oauth.endpoint value set.
	Make sure you correctly specify the endpoint in your custom resource.
	`

	errorProtocolIsMissingOauthEndpoint = `The EdgeConnect's specification has an invalid oauth.endpoint value set.
	Should include 'https' protocol.
	Make sure you correctly specify the oauth.endpoint in your custom resource.
	`

	errorUnknownSSOServer = `The EdgeConnect's specification has an invalid oauth.endpoint value set.
	Example valid values:
	-  For prod: "sso.dynatrace.com"
	-  For dev: "sso-dev.dynatracelabs.com",
	-  For sprint: "sso.sprint.dynatracelabs.com"
	Make sure you correctly specify the endpoint in your custom resource.
	`
)

var (
	allowedSSODomains = []string{
		"sso-dev.dynatracelabs.com",
		"sso.sprint.dynatracelabs.com",
		"sso.dynatrace.com",
	}
)

func isValidSSOServerURL(ctx context.Context, _ *Validator, ec *edgeconnect.EdgeConnect) string {
	_, err := url.Parse(ec.Spec.OAuth.Endpoint)
	if err != nil {
		return errorInvalidOauthEndpoint
	}

	return ""
}

func checkSSOServerProtocol(ctx context.Context, _ *Validator, ec *edgeconnect.EdgeConnect) string {
	url, err := url.Parse(ec.Spec.OAuth.Endpoint)
	if err != nil {
		return ""
	}

	if strings.ToLower(url.Scheme) != "https" {
		return errorProtocolIsMissingOauthEndpoint
	}

	return ""
}

func isAllowedSSOServer(ctx context.Context, _ *Validator, ec *edgeconnect.EdgeConnect) string {
	url, err := url.Parse(ec.Spec.OAuth.Endpoint)
	if err != nil {
		return ""
	}

	if !slices.Contains(allowedSSODomains, strings.ToLower(url.Host)) {
		return errorUnknownSSOServer
	}

	return ""
}
