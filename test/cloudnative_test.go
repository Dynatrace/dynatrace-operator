//go:build e2e
// +build e2e

package test

import (
	"github.com/Dynatrace/dynatrace-operator/test/namespace"
	"testing"
)

const (
	dynatraceNamespace = "dynatrace"
)

func TestMain(m *testing.M) {
	environment := getEnvironment()
	environment.Setup(namespace.Create(dynatraceNamespace))

	environment.Finish(namespace.Delete(dynatraceNamespace))
	environment.Run(m)
}

func TestATest(t *testing.T) {
	println("")
}
