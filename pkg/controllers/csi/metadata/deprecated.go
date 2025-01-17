package metadata

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// only kept for migration
func TenantUUIDFromApiUrl(apiUrl string) (string, error) {
	parsedUrl, err := url.Parse(apiUrl)
	if err != nil {
		return "", errors.WithMessagef(err, "problem parsing tenant id from url %s", apiUrl)
	}

	// Path = "/e/<token>/api" -> ["e",  "<tenant>", "api"]
	subPaths := strings.FieldsFunc(parsedUrl.Path, runeIs('/'))
	if len(subPaths) >= 3 && subPaths[0] == "e" && subPaths[2] == "api" {
		return subPaths[1], nil
	}

	hostnameWithDomains := strings.FieldsFunc(parsedUrl.Hostname(), runeIs('.'))
	if len(hostnameWithDomains) >= 1 {
		return hostnameWithDomains[0], nil
	}

	return "", errors.Errorf("problem getting tenant id from API URL '%s'", apiUrl)
}

func runeIs(wanted rune) func(rune) bool {
	return func(actual rune) bool {
		return actual == wanted
	}
}
