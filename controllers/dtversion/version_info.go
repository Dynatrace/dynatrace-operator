package dtversion

import (
	"fmt"
	"regexp"
	"strconv"
)

type VersionInfo struct {
	major   int
	minor   int
	release int
}

var versionRegex = regexp.MustCompile(`^([\d]+)\.([\d]+)\.([\d]+)`)

// CompareVersionInfo returns:
// 	0: if a == b
//  n > 0: if a > b
//  n < 0: if a < b
//  0 with error: if a == nil || b == nil
func CompareVersionInfo(a VersionInfo, b VersionInfo) int {
	// Check major version
	result := a.major - b.major
	if result != 0 {
		return result
	}

	// Major is equal, check minor
	result = a.minor - b.minor
	if result != 0 {
		return result
	}

	// Major and minor is equal, check release
	result = a.release - b.release
	return result
}

func ExtractVersion(versionString string) (VersionInfo, error) {
	version := versionRegex.FindStringSubmatch(versionString)

	if len(version) < 4 {
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

	return VersionInfo{major, minor, release}, nil
}

func (v VersionInfo) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.release)
}
