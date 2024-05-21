package troubleshoot

import (
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
)

type component string

const (
	componentOneAgent    component = "OneAgent"
	componentCodeModules component = "OneAgentCodeModules"
	componentActiveGate  component = "ActiveGate"

	customImagePostfix = " (custom image)"
)

func (c component) String() string {
	return string(c)
}

func (c component) Name(isCustomImage bool) string {
	if isCustomImage {
		return c.String() + customImagePostfix
	}

	return c.String()
}

func (c component) SkipImageCheck(image string) bool {
	return image == "" && c != componentCodeModules
}

func (c component) getImage(dynakube *dynatracev1beta2.DynaKube) (string, bool) {
	if dynakube == nil {
		return "", false
	}

	switch c {
	case componentOneAgent:
		if dynakube.CustomOneAgentImage() != "" {
			return dynakube.CustomOneAgentImage(), true
		}

		return dynakube.OneAgentImage(), false
	case componentCodeModules:
		return dynakube.CustomCodeModulesImage(), true
	case componentActiveGate:
		if dynakube.CustomActiveGateImage() != "" {
			return dynakube.CustomActiveGateImage(), true
		}

		return dynakube.ActiveGateImage(), false
	}

	return "", false
}
