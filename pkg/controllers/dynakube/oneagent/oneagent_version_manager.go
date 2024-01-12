package oneagent

import (
	"github.com/open-feature/go-sdk/openfeature"
)

const (
	MinOneAgentVersionSupported = "1.2.3" // minimum version supported by this operator installer
	MaxOneAgentVersionSupported = "1.4.5" // maximum version supported by this operator installer
)

func NewOneAgentVersionManager(provider openfeature.FeatureProvider) *VersionManager {

	err := openfeature.SetProvider(provider)
	if err != nil {
		// handle error
	}
	/*initialize supported version ranges */
	client := openfeature.NewClient("dynakube-operator")
	return &VersionManager{
		OfClient: client,
	}
}

// bad name, but manager is pretty generic
type VersionManager struct {
	OfClient *openfeature.Client
}

func (vm *VersionManager) IsOneAgentVersionSupported(oneAgentVersion string) bool {
	// do semver version comparison of oneAgentVersion and
	// * min/max version supported by this operator, and also get
	// * versions supported by the ConfigMap
	// to see if OneAgent version that should be installed is supported
	return true
}

// we could use the EvaluationContext to map the featureflags of each version- there has to be a mapping
// of version ranges to configurations
func (vm *VersionManager) GetContextForVersion(versoneAgentVersion string) openfeature.EvaluationContext {
	return openfeature.EvaluationContext{}
}
