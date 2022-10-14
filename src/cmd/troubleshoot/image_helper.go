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

// Split image path in into its components.
// Some image path examples that work with this function:
//
//	aaa00000.dynatrace.com/linux/oneagent
//	aaa00000.dynatrace.com/linux/activegate:1.123
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
		logInfof("using latest image version")
		return fields[0], "latest", nil
	}

	if len(fields) >= 2 {
		logInfof("using custom image version: %s", fields[1])
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

func getOneAgentImageEndpoint(troubleshootCtx *troubleshootContext) string {
	imageEndpoint := ""

	apiEndpoint := removeSchemaRegex.FindStringSubmatch(troubleshootCtx.dynakube.Spec.APIURL)
	registry := removeApiEndpointRegex.FindStringSubmatch(apiEndpoint[1])
	imageEndpoint = registry[1] + "/linux/oneagent"

	customImage := troubleshootCtx.dynakube.CustomOneAgentImage()
	version := troubleshootCtx.dynakube.Version()

	if customImage != "" {
		imageEndpoint = customImage
	} else if version != "" {
		imageEndpoint = imageEndpoint + ":" + version
	}

	logInfof("OneAgent image endpoint '%s'", imageEndpoint)
	return imageEndpoint
}

func getOneAgentCodeModulesImageEndpoint(troubleshootCtx *troubleshootContext) string {
	imageEndpoint := troubleshootCtx.dynakube.CodeModulesImage()

	if imageEndpoint != "" {
		logInfof("OneAgent codeModules image endpoint '%s'", imageEndpoint)
	} else {
		logInfof("OneAgent codeModules image endpoint is not used.")
	}
	return imageEndpoint
}

func getActiveGateImageEndpoint(troubleshootCtx *troubleshootContext) string {
	imageEndpoint := ""

	apiEndpoint := removeSchemaRegex.FindStringSubmatch(troubleshootCtx.dynakube.Spec.APIURL)
	registry := removeApiEndpointRegex.FindStringSubmatch(apiEndpoint[1])
	imageEndpoint = registry[1] + "/linux/activegate"

	customActiveGateImage := troubleshootCtx.dynakube.CustomActiveGateImage()
	if customActiveGateImage != "" {
		imageEndpoint = customActiveGateImage
	}

	logInfof("ActiveGate image endpoint '%s'", imageEndpoint)
	return imageEndpoint
}
