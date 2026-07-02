// Package dtapiurl provides helpers for working with Dynatrace API URLs,
// in particular for mapping between 2nd gen and 3rd gen URLs.
package dtapiurl

import (
	"net/url"
	"strings"
)

const thirdGenAppsHostParts = 2

// isThirdGen reports whether the given hostname belongs to a 3rd gen environment,
// i.e. it contains the ".apps." segment.
func isThirdGen(hostname string) bool {
	return strings.Contains(hostname, ".apps.")
}

// mapToSecondGen remaps a 3rd gen URL (*.apps.*) to its 2nd gen equivalent in place
// and sets its path to "/api". URLs that are not 3rd gen are left untouched.
func mapToSecondGen(u *url.URL) {
	hostname := u.Hostname()

	parts := strings.SplitN(hostname, ".apps.", thirdGenAppsHostParts)
	if len(parts) != thirdGenAppsHostParts {
		return
	}

	prefix, suffix := parts[0], parts[1]

	var newHostname string

	if !strings.Contains(prefix, ".") {
		newHostname = prefix + ".live." + suffix
	} else {
		newHostname = prefix + "." + suffix
	}

	if port := u.Port(); port != "" {
		u.Host = newHostname + ":" + port
	} else {
		u.Host = newHostname
	}

	u.Path = "/api"
}

// ToSecondGen returns the 2nd gen equivalent of the given API URL string.
// A 3rd gen URL (*.apps.*) is remapped to its 2nd gen equivalent (including the
// "/api" path), while a 2nd gen URL is returned unchanged. If the input cannot be
// parsed, it is returned as-is.
func ToSecondGen(apiURL string) string {
	u, err := url.Parse(apiURL)
	if err != nil {
		return apiURL
	}

	if !isThirdGen(u.Hostname()) {
		return apiURL
	}

	mapToSecondGen(u)

	return u.String()
}
