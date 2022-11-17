package troubleshoot

import "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"

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

func (c component) getImage(dynakube *v1beta1.DynaKube) (string, bool) {
	if dynakube == nil {
		return "", false
	}

	switch c {
	case componentOneAgent:
		return dynakube.OneAgentImage(), dynakube.CustomOneAgentImage() != ""
	case componentCodeModules:
		return dynakube.CodeModulesImage(), false
	case componentActiveGate:
		return dynakube.ActiveGateImage(), dynakube.ActiveGateImage() != ""
	}
	return "", false
}
