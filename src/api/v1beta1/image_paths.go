package v1beta1

import (
	"fmt"
	"strings"
)

type imagePathResolver interface {
	CustomImagePath() string
	DefaultImagePath() string

	apiUrl() string
}

type genericImagePath struct {
	dynaKube *DynaKube
}

func newGenericImagePath(dynaKube *DynaKube) *genericImagePath {
	return &genericImagePath{
		dynaKube: dynaKube,
	}
}

func (imagePath *genericImagePath) apiUrl() string {
	return imagePath.dynaKube.Spec.APIURL
}

var _ imagePathResolver = (*activeGateImagePath)(nil)

type activeGateImagePath genericImagePath

func newActiveGateImagePath(dynaKube *DynaKube) *activeGateImagePath {
	return (*activeGateImagePath)(newGenericImagePath(dynaKube))
}

func (imagePath *activeGateImagePath) apiUrl() string {
	return (*genericImagePath)(imagePath).apiUrl()
}

func (imagePath *activeGateImagePath) CustomImagePath() string {
	dk := imagePath.dynaKube
	if dk.DeprecatedActiveGateMode() {
		if dk.Spec.KubernetesMonitoring.Image != "" {
			return dk.Spec.KubernetesMonitoring.Image
		} else if dk.Spec.Routing.Image != "" {
			return dk.Spec.Routing.Image
		}
	} else if dk.ActiveGateMode() {
		if dk.Spec.ActiveGate.Image != "" {
			return dk.Spec.ActiveGate.Image
		}
	}

	return ""
}

func (imagePath *activeGateImagePath) DefaultImagePath() string {
	return "linux/activegate:latest"
}

var _ imagePathResolver = (*statsdImagePath)(nil)

type statsdImagePath genericImagePath

func newStatsdImagePath(dynaKube *DynaKube) *statsdImagePath {
	return (*statsdImagePath)(newGenericImagePath(dynaKube))
}

func (imagePath *statsdImagePath) apiUrl() string {
	return (*genericImagePath)(imagePath).apiUrl()
}

func (imagePath *statsdImagePath) CustomImagePath() string {
	if imagePath.dynaKube.NeedsStatsd() {
		return imagePath.dynaKube.FeatureCustomStatsdImage()
	}
	return ""
}

func (imagePath *statsdImagePath) DefaultImagePath() string {
	return "linux/dynatrace-datasource-statsd:latest"
}

var _ imagePathResolver = (*eecImagePath)(nil)

type eecImagePath genericImagePath

func newEecImagePath(dynaKube *DynaKube) *eecImagePath {
	return (*eecImagePath)(newGenericImagePath(dynaKube))
}

func (imagePath *eecImagePath) apiUrl() string {
	return (*genericImagePath)(imagePath).apiUrl()
}

func (imagePath *eecImagePath) CustomImagePath() string {
	if imagePath.dynaKube.NeedsStatsd() {
		return imagePath.dynaKube.FeatureCustomEecImage()
	}
	return ""
}

func (imagePath *eecImagePath) DefaultImagePath() string {
	return "linux/dynatrace-eec:latest"
}

func resolveImagePath(resolver imagePathResolver) string {
	customImage := resolver.CustomImagePath()
	if len(customImage) > 0 {
		return customImage
	}

	apiUrl := resolver.apiUrl()
	if apiUrl == "" {
		return ""
	}

	registry := buildImageRegistry(apiUrl)
	return fmt.Sprintf("%s/%s", registry, resolver.DefaultImagePath())
}

func buildImageRegistry(apiURL string) string {
	registry := strings.TrimPrefix(apiURL, "https://")
	registry = strings.TrimPrefix(registry, "http://")
	registry = strings.TrimSuffix(registry, "/api")
	return registry
}
