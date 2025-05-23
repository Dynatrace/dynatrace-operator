package troubleshoot

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
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

func (c component) getImage(dk *dynakube.DynaKube) (string, bool) {
	if dk == nil {
		return "", false
	}

	switch c {
	case componentOneAgent:
		if dk.OneAgent().GetCustomImage() != "" {
			return dk.OneAgent().GetCustomImage(), true
		}

		return dk.OneAgent().GetImage(), false
	case componentCodeModules:
		return dk.OneAgent().GetCustomCodeModulesImage(), true
	case componentActiveGate:
		if dk.ActiveGate().GetCustomImage() != "" {
			return dk.ActiveGate().GetCustomImage(), true
		}

		return dk.ActiveGate().GetImage(), false
	}

	return "", false
}
