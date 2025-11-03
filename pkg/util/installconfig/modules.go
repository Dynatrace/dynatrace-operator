package installconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	ModulesJSONEnv = "modules.json"

	validationErrorTemplate = "%s has been disabled during Operator install. The necessary resources for %s to work are not present on the cluster. Redeploy the Operator via Helm with all the necessary resources enabled."

	validationDependentErrorTemplate = "%s has been disabled during Operator install. That is a dependency for %s to work. Redeploy the Operator via Helm with all the necessary resources enabled."
)

var (
	once sync.Once

	modules Modules

	// needed for testing
	override *Modules

	fallbackModules = Modules{
		CSIDriver:            true,
		ActiveGate:           true,
		OneAgent:             true,
		Extensions:           true,
		LogMonitoring:        true,
		EdgeConnect:          true,
		Supportability:       true,
		KSPM:                 true,
		KubernetesMonitoring: true,
	}

	log = logd.Get().WithName("install-config")
)

type Modules struct {
	CSIDriver            bool `json:"csiDriver"`
	ActiveGate           bool `json:"activeGate"`
	OneAgent             bool `json:"oneAgent"`
	Extensions           bool `json:"extensions"`
	LogMonitoring        bool `json:"logMonitoring"`
	EdgeConnect          bool `json:"edgeConnect"`
	Supportability       bool `json:"supportability"`
	KubernetesMonitoring bool `json:"kubernetesMonitoring"`
	KSPM                 bool `json:"kspm"`
}

func GetModules() Modules {
	if override != nil {
		return *override
	}

	ReadModules()

	return modules
}

func ReadModules() {
	ReadModulesToLogger(log)
}

func ReadModulesToLogger(log logd.Logger) {
	once.Do(func() {
		modulesJSON := os.Getenv(ModulesJSONEnv)
		if modulesJSON == "" {
			log.Info("envvar not set, using default", "envvar", ModulesJSONEnv)

			modules = fallbackModules
		}

		err := json.Unmarshal([]byte(modulesJSON), &modules)
		if err != nil {
			log.Info("problem unmarshalling envvar content, using default", "envvar", ModulesJSONEnv, "err", err)

			modules = fallbackModules
		}

		log.Info("envvar content read and set", "envvar", ModulesJSONEnv, "value", modulesJSON)
	})
}

// SetModulesOverride is a testing function, so you can easily unittest function using the GetModules() func
func SetModulesOverride(t *testing.T, modules Modules) {
	t.Helper()

	override = &modules

	t.Cleanup(func() {
		override = nil
	})
}

func GetModuleValidationErrorMessage(moduleName string) string {
	return fmt.Sprintf(validationErrorTemplate, moduleName, moduleName)
}

func GetDependentModuleValidationErrorMessage(moduleName, dependentModule string) string {
	return fmt.Sprintf(validationDependentErrorTemplate, moduleName, dependentModule)
}
