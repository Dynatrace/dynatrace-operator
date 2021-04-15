package dtversion

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type VersionInfo struct {
	major     int
	minor     int
	release   int
	timestamp string
}

var versionRegex = regexp.MustCompile(`^([\d]+)\.([\d]+)\.([\d]+)\.([\d]+\-[\d]+)$`)

// CompareVersionInfo returns:
// 	0: if a == b
//  n > 0: if a > b
//  n < 0: if a < b
//  0 with error: if a == nil || b == nil
func CompareVersionInfo(a VersionInfo, b VersionInfo) int {
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

func ExtractVersion(versionString string) (VersionInfo, error) {
	version := versionRegex.FindStringSubmatch(versionString)

	if len(version) < 5 {
		return VersionInfo{}, fmt.Errorf("version malformed: %s", versionString)
	}

	major, err := strconv.Atoi(version[1])
	if err != nil {
		return VersionInfo{}, err
	}

	minor, err := strconv.Atoi(version[2])
	if err != nil {
		return VersionInfo{}, err
	}

	release, err := strconv.Atoi(version[3])
	if err != nil {
		return VersionInfo{}, err
	}

	return VersionInfo{major, minor, release, version[4]}, nil
}

func (v VersionInfo) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.release)
}

// NeedsUpgradeRaw parses prev and curr, and returns true when curr is a newer version than prev, or false if they are
// the same. In case curr is older than prev an error is returned.
func NeedsUpgradeRaw(prev string, curr string) (bool, error) {
	parsedPrev, err := ExtractVersion(prev)
	if err != nil {
		return false, errors.WithMessage(err, "failed to parse version")
	}

	parsedCurr, err := ExtractVersion(curr)
	if err != nil {
		return false, errors.WithMessage(err, "failed to parse version")
	}

	comp := CompareVersionInfo(parsedPrev, parsedCurr)
	if comp > 0 {
		return false, errors.Errorf("trying to downgrade from '%s' to '%s'", parsedPrev, parsedCurr)
	}

	return comp < 0, nil
}
