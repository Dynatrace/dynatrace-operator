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

func splitImageName(imageName string) (imageInfo, error) {
	// some image path examples that work with this function
	//   aaa00000.dynatrace.com/linux/oneagent
	//   aaa00000.dynatrace.com/linux/activegate:1.123

	imgInfo := imageInfo{}

	registryMatches := registryRegex.FindStringSubmatch(imageName)
	if len(registryMatches) < 2 {
		return imageInfo{}, fmt.Errorf("invalid image - registry not found (%s)", imageName)
	}
	imgInfo.registry = registryRegex.FindStringSubmatch(imageName)[1]

	imageMatches := imageRegex.FindStringSubmatch(imageName)
	if len(imageMatches) < 2 {
		return imageInfo{}, fmt.Errorf("invalid image - endpoint not found (%s)", imageName)
	}
	imgInfo.image = imageRegex.FindStringSubmatch(imageName)[1]

	imgInfo.version = ""

	// check if image has version set
	fields := strings.Split(imgInfo.image, ":")
	if len(fields) == 1 || len(fields) >= 2 && fields[1] == "" {
		// no version set, default to latest
		imgInfo.version = "latest"
		logInfof("using latest image version")
	} else if len(fields) >= 2 {
		imgInfo.image = fields[0]
		imgInfo.version = fields[1]
		logInfof("using custom image version: %s", imgInfo.version)
	} else {
		return imageInfo{}, fmt.Errorf("invalid version of the image {\"image\": \"%s\"}", imgInfo.image)
	}
	return imgInfo, nil
}

func splitCustomImageName(imageURL string) (imageInfo, error) {
	imgInfo := imageInfo{}

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

	sr := removeSchemaRegex.FindStringSubmatch(troubleshootCtx.dynakube.Spec.APIURL)
	er := removeApiEndpointRegex.FindStringSubmatch(sr[1])
	imageEndpoint = er[1] + "/linux/oneagent"

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

	sr := removeSchemaRegex.FindStringSubmatch(troubleshootCtx.dynakube.Spec.APIURL)
	er := removeApiEndpointRegex.FindStringSubmatch(sr[1])
	imageEndpoint = er[1] + "/linux/activegate"

	customActiveGateImage := troubleshootCtx.dynakube.CustomActiveGateImage()
	if customActiveGateImage != "" {
		imageEndpoint = customActiveGateImage
	}

	logInfof("ActiveGate image endpoint '%s'", imageEndpoint)
	return imageEndpoint
}
