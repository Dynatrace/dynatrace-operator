package mapper

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
)

const ignoredNsEnv = "IGNORED_NAMESPACES"

var (
	once sync.Once
	ignoredNamespaces []string
)


type IgnoredError struct {
	namespace string
}

func (err IgnoredError) Error() string {
	return fmt.Sprintf("no dynakube can match the namespace(%s), because it is ignored according to the install", err.namespace)
}

func IsIgnoredNamespace(namespace string) bool {
	once.Do(parseIgnoreEnv)

	for _, pattern := range ignoredNamespaces {
		if matched, _ := regexp.MatchString(pattern, namespace); matched {
			return true
		}
	}

	return false

}

func parseIgnoreEnv() {
	ignoredNs, ok := os.LookupEnv(ignoredNsEnv)
	if !ok {
		ignoredNamespaces = defaultIgnoredNamespaces()
	} else {
		err := json.Unmarshal([]byte(ignoredNs), &ignoredNamespaces)
		if err != nil {
			log.Error(err, "failed to parse ignored namespaces env, using defaults", "env", ignoredNsEnv, "value", ignoredNs)
			ignoredNamespaces = defaultIgnoredNamespaces()
		}
		log.Info("parsed ignored namespaces", "value", ignoredNamespaces)
	}
}

func defaultIgnoredNamespaces() []string {
	defaultIgnoredNamespaces := []string{
		env.DefaultNamespace(),
		"^kube-.*",
		"^openshift(-.*)?",
		"^gke-.*",
		"^gmp-.*",
	}

	return defaultIgnoredNamespaces
}
