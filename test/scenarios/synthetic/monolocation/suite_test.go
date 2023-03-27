//go:build e2e

package monolocation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var suiteEnvironment env.Environment

func TestMain(m *testing.M) {
	suiteEnvironment = environment.Get()
	suiteEnvironment.Run(m)
}

func TestSyntheticWithSingleLocation(t *testing.T) {
	suiteEnvironment.Test(t, newFeature(t))
}
