package troubleshoot

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	removeSchemaRegex      = regexp.MustCompile("^.*//(.*)$")
	removeApiEndpointRegex = regexp.MustCompile("^(.*)/[^/]*$")
	registryRegex          = regexp.MustCompile(`^(.*)/linux.*$`)
	imageRegex             = regexp.MustCompile(`^.*/(linux.*)$`)
)

type imageInfo struct {
	registry string
	image    string
	version  string
}

func (codeModulesImageInfo imageInfo) imageUrl() string {
	return fmt.Sprintf("https://%s/%s", codeModulesImageInfo.registry, codeModulesImageInfo.image)
}

// splitImageName splits an image path and returns an imageInfo instance
// containing the referenced registry, image and version.
// Some image path examples that work with this function:
//
// * aaa00000.dynatrace.com/linux/oneagent
// * aaa00000.dynatrace.com/linux/activegate:1.123
func splitImageName(imageName string) (imageInfo, error) {
	imgInfo := imageInfo{}

	registryMatches := registryRegex.FindStringSubmatch(imageName)
	if len(registryMatches) < 2 {
		return imageInfo{}, fmt.Errorf("invalid image - registry not found (%s)", imageName)
	}
	imgInfo.registry = registryMatches[1]

	imageMatches := imageRegex.FindStringSubmatch(imageName)
	if len(imageMatches) < 2 {
		return imageInfo{}, fmt.Errorf("invalid image - endpoint not found (%s)", imageName)
	}
	imgInfo.image = imageMatches[1]

	imgInfo.version = ""

	// check if image has version set
	var err error
	imgInfo.image, imgInfo.version, err = parseImageVersion(imgInfo.image)
	if err != nil {
		return imageInfo{}, err
	}
	return imgInfo, nil
}

func parseImageVersion(image string) (string, string, error) {
	fields := strings.Split(image, ":")

	if len(fields) == 1 || len(fields) >= 2 && fields[1] == "" {
		return fields[0], "latest", nil
	}

	if len(fields) >= 2 {
		return fields[0], fields[1], nil
	}

	return "", "", fmt.Errorf("invalid version of the image {\"image\": \"%s\"}", image)
}

func splitCustomImageName(imageURL string) (imageInfo, error) {
	imgInfo := imageInfo{}

	// extract registry (not-greedy until first '/') and image name
	customImageRegistryRegex := regexp.MustCompile(`^(.*?)/(.*)$`)

	registryMatches := customImageRegistryRegex.FindStringSubmatch(imageURL)
	if len(registryMatches) < 3 {
		return imageInfo{}, fmt.Errorf("invalid image path (%s) - could not parse", imageURL)
	}
	imgInfo.registry = registryMatches[1]
	imgInfo.image = registryMatches[2]

	if len(imgInfo.registry) == 0 {
		return imageInfo{}, fmt.Errorf("invalid image path (%s) - registry not found", imageURL)
	}
	if len(imgInfo.image) == 0 {
		return imageInfo{}, fmt.Errorf("invalid image path (%s) - image path not found", imageURL)
	}

	return imgInfo, nil
}
