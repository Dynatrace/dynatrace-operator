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

func splitImageName(imageName string) (registry string, image string, version string, err error) {
	// some image path examples that work with this function
	//   exq67461.dev.dynatracelabs.com/linux/oneagent
	//   exq67461.dev.dynatracelabs.com/linux/activegate:1.123

	err = nil

	registryMatches := registryRegex.FindStringSubmatch(imageName)
	if len(registryMatches) < 2 {
		err = fmt.Errorf("invalid image - registry not found (%s)", imageName)
		return
	}
	registry = registryRegex.FindStringSubmatch(imageName)[1]

	imageMatches := imageRegex.FindStringSubmatch(imageName)
	if len(imageMatches) < 2 {
		err = fmt.Errorf("invalid image - endpoint not found (%s)", imageName)
		return
	}
	image = imageRegex.FindStringSubmatch(imageName)[1]

	version = ""

	// check if image has version set
	fields := strings.Split(image, ":")
	if len(fields) == 1 || len(fields) >= 2 && fields[1] == "" {
		// no version set, default to latest
		version = "latest"
		logInfof("using latest image version")
	} else if len(fields) >= 2 {
		image = fields[0]
		version = fields[1]
		logInfof("using custom image version: %s", version)
	} else {
		err = fmt.Errorf("invalid version of the image {\"image\": \"%s\"}", image)
	}
	return
}

func splitCustomImageName(imageURL string) (registry string, imagePath string, err error) {
	customImageRegistryRegex := regexp.MustCompile(`^(.*?)/(.*)$`)

	err = nil

	registryMatches := customImageRegistryRegex.FindStringSubmatch(imageURL)
	if len(registryMatches) < 3 {
		err = fmt.Errorf("invalid image path (%s) - could not parse", imageURL)
		return
	}
	registry = registryMatches[1]
	imagePath = registryMatches[2]

	if len(registry) == 0 {
		err = fmt.Errorf("invalid image path (%s) - registry not found", imageURL)
		return
	}
	if len(imagePath) == 0 {
		err = fmt.Errorf("invalid image path (%s) - image path not found", imageURL)
		return
	}
	return
}

func getOneAgentImageEndpoint(troubleshootCtx *troubleshootContext) string {
	customImage := ""
	imageEndpoint := ""
	version := ""

	sr := removeSchemaRegex.FindStringSubmatch(troubleshootCtx.dynakube.Spec.APIURL)
	er := removeApiEndpointRegex.FindStringSubmatch(sr[1])
	imageEndpoint = er[1] + "/linux/oneagent"

	if troubleshootCtx.dynakube.ClassicFullStackMode() {
		customImage = troubleshootCtx.dynakube.Spec.OneAgent.ClassicFullStack.Image
		version = troubleshootCtx.dynakube.Spec.OneAgent.ClassicFullStack.Version
	} else if troubleshootCtx.dynakube.CloudNativeFullstackMode() {
		customImage = troubleshootCtx.dynakube.Spec.OneAgent.CloudNativeFullStack.Image
		version = troubleshootCtx.dynakube.Spec.OneAgent.CloudNativeFullStack.Version
	} else if troubleshootCtx.dynakube.HostMonitoringMode() {
		customImage = troubleshootCtx.dynakube.Spec.OneAgent.HostMonitoring.Image
		version = troubleshootCtx.dynakube.Spec.OneAgent.HostMonitoring.Version
	}

	if customImage != "" {
		imageEndpoint = customImage
	} else if version != "" {
		imageEndpoint = imageEndpoint + ":" + version
	}

	logInfof("OneAgent image endpoint '%s'", imageEndpoint)
	return imageEndpoint
}

func getOneAgentCodeModulesImageEndpoint(troubleshootCtx *troubleshootContext) string {
	imageEndpoint := ""
	switch {
	case troubleshootCtx.dynakube.CloudNativeFullstackMode():
		imageEndpoint = troubleshootCtx.dynakube.Spec.OneAgent.CloudNativeFullStack.CodeModulesImage

	case troubleshootCtx.dynakube.ApplicationMonitoringMode():
		imageEndpoint = troubleshootCtx.dynakube.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage
	}

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

	if troubleshootCtx.dynakube.Spec.ActiveGate.Image != "" {
		imageEndpoint = troubleshootCtx.dynakube.Spec.ActiveGate.Image
	}

	logInfof("ActiveGate image endpoint '%s'", imageEndpoint)
	return imageEndpoint
}
