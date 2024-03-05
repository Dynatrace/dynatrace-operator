// Package dtversion's purpose is to convert the component/image versions used by Dynatrace into semver
// - Example version used by Dynatrace:
//   - 1.283.132.20240205-143805 (this translates to [version-number].[sprint-number].[quick-fix].[build-date]-[build-timestamp])
//   - The operator does not care about the [build-date] and [build-timestamp] parts, so we can ignore those, and after that converting it to semver is trivial
//   - As semver: v1.283.132
package dtversion

import (
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
)

const (
	semverPrefix = "v"
	semverSep    = "."

	// relevantVersionLength defines how many segments of the semver we care about, for-example:
	// in case of 3, we only care about major.minor.patch and the rest is ignored.
	relevantSemverLength = 3
)

func ToSemver(version string) (string, error) {
	if version == "" {
		return "", nil
	}

	split := strings.Split(version, semverSep)

	var semantic string

	if len(split) > relevantSemverLength {
		semantic = strings.Join(split[:relevantSemverLength], semverSep)
	} else {
		semantic = strings.Join(split, semverSep)
	}

	if !strings.HasPrefix(semantic, semverPrefix) {
		semantic = semverPrefix + semantic
	}

	result := semver.Canonical(semantic)

	if !semver.IsValid(result) {
		return "", errors.New(version + " is not possible to convert to a semver format.")
	}

	return result, nil
}
