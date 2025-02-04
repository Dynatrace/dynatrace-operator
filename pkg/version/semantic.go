package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type SemanticVersion struct {
	timestamp string
	major     int
	minor     int
	release   int
}

var versionRegex = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)\.(\d+-\d+)$`)

// Max sub match = orignal string + 4 groups from versionRegex ^.
const maxStringSubMatch = 5

// CompareSemanticVersions returns:
//
//		0: if a == b
//	 n > 0: if a > b
//	 n < 0: if a < b
//	 0 with error: if a == nil || b == nil
func CompareSemanticVersions(a SemanticVersion, b SemanticVersion) int {
	if a.major != b.major {
		return a.major - b.major
	}

	if a.minor != b.minor {
		return a.minor - b.minor
	}

	if a.release != b.release {
		return a.release - b.release
	}

	return strings.Compare(a.timestamp, b.timestamp)
}

func ExtractSemanticVersion(versionString string) (SemanticVersion, error) {
	version := versionRegex.FindStringSubmatch(versionString)

	if len(version) < maxStringSubMatch {
		return SemanticVersion{}, fmt.Errorf("version malformed: %s", versionString)
	}

	major, err := strconv.Atoi(version[1])
	if err != nil {
		return SemanticVersion{}, err
	}

	minor, err := strconv.Atoi(version[2])
	if err != nil {
		return SemanticVersion{}, err
	}

	release, err := strconv.Atoi(version[3])
	if err != nil {
		return SemanticVersion{}, err
	}

	return SemanticVersion{major: major, minor: minor, release: release, timestamp: version[4]}, nil
}

func (version SemanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d.%s", version.major, version.minor, version.release, version.timestamp)
}

// IsDowngrade parses prev and curr, and returns true when curr is a older version than prev.
func IsDowngrade(prev string, curr string) (bool, error) {
	parsedPrev, err := ExtractSemanticVersion(prev)
	if err != nil {
		return false, errors.WithMessage(err, "failed to parse version")
	}

	parsedCurr, err := ExtractSemanticVersion(curr)
	if err != nil {
		return false, errors.WithMessage(err, "failed to parse version")
	}

	comp := CompareSemanticVersions(parsedPrev, parsedCurr)

	return comp > 0, nil
}
