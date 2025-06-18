package metadata

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// only kept for migration
func TenantUUIDFromAPIURL(apiURL string) (string, error) {
	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		return "", errors.WithMessagef(err, "problem parsing tenant id from url %s", apiURL)
	}

	// Path = "/e/<token>/api" -> ["e",  "<tenant>", "api"]
	subPaths := strings.FieldsFunc(parsedURL.Path, runeIs('/'))
	if len(subPaths) >= 3 && subPaths[0] == "e" && subPaths[2] == "api" {
		return subPaths[1], nil
	}

	hostnameWithDomains := strings.FieldsFunc(parsedURL.Hostname(), runeIs('.'))
	if len(hostnameWithDomains) >= 1 {
		return hostnameWithDomains[0], nil
	}

	return "", errors.Errorf("problem getting tenant id from API URL '%s'", apiURL)
}

func runeIs(wanted rune) func(rune) bool {
	return func(actual rune) bool {
		return actual == wanted
	}
}
