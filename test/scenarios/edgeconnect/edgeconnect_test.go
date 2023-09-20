//go:build e2e

package edgeconnect

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.GetStandardKubeClusterEnvironment()
	testEnvironment.Run(m)
}

func TestEdgeConnect(t *testing.T) {
	testEnvironment.Test(t, install(t))
}
