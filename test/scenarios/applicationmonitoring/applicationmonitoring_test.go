//go:build e2e

package applicationmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.Run(m)
}

func TestApplicationMonitoring(t *testing.T) {
	testEnvironment.Test(t, dataIngest(t))
}

func TestLabelVersionDetection(t *testing.T) {
	testEnvironment.Test(t, labelVersionDetection(t))
}

func TestReadOnlyCSIVolume(t *testing.T) {
	testEnvironment.Test(t, readOnlyCSIVolume(t))
}
