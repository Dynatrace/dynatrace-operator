package troubleshoot

import "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"

type component string

const (
	oneAgent           component = "OneAgent"
	codeModules        component = "OneAgentCodeModules"
	activeGate         component = "ActiveGate"
	customImagePostfix component = " (custom image)"
)

func (c component) String() string {
	return string(c)
}

func (c component) Name(isCustomImage bool) string {
	if isCustomImage {
		return c.String() + customImagePostfix.String()
	}
	return c.String()
}

func (c component) getImage(dynakube *v1beta1.DynaKube) (string, bool) {
	if dynakube == nil {
		return "", false
	}

	switch c {
	case oneAgent:
		return dynakube.OneAgentImage(), dynakube.CustomOneAgentImage() != ""
	case codeModules:
		return dynakube.CodeModulesImage(), false
	case activeGate:
		return dynakube.ActiveGateImage(), dynakube.ActiveGateImage() != ""
	}
	return "", false
}
