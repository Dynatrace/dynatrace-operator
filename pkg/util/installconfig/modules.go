package installconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	modulesJsonEnv = "modules.json"

	validationErrorTemplate = "%s has been disabled during Operator install. The necessary resources for %s to work are not present on the cluster. Redeploy the Operator via Helm with all the necessary resources enabled."
)

var (
	once sync.Once

	modules Modules

	fallbackModules = Modules{
		ActiveGate:  true,
		OneAgent:    true,
		Extensions:  true,
		LogModule:   true,
		EdgeConnect: true,
	}

	log = logd.Get().WithName("install-config")
)

type Modules struct {
	ActiveGate  bool `json:"activeGate"`
	OneAgent    bool `json:"oneAgent"`
	Extensions  bool `json:"extensions"`
	LogModule   bool `json:"logModule"`
	EdgeConnect bool `json:"edgeConnect"`
}

func GetModules() Modules {
	once.Do(func() {
		modulesJson := os.Getenv(modulesJsonEnv)
		if modulesJson == "" {
			log.Info("envvar not set, using default", "envvar", modulesJsonEnv)

			modules = fallbackModules
		}

		err := json.Unmarshal([]byte(modulesJson), &modules)
		if err != nil {
			log.Info("problem unmarshalling envvar content, using default", "envvar", modulesJsonEnv, "err", err)

			modules = fallbackModules
		}

		log.Info("envvar content read and set", "envvar", modulesJsonEnv, "value", modulesJson)
	})

	return modules
}

func GetModuleValidationErrorMessage(moduleName string) string {
	return fmt.Sprintf(validationErrorTemplate, moduleName, moduleName)
}
