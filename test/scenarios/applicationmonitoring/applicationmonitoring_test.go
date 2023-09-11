//go:build e2e

package applicationmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	log.SetLogger(logger.Factory.GetLogger("e2e-application-monitoring"))
	testEnvironment = environment.Get()
	testEnvironment.Run(m)
}

func TestDataIngest(t *testing.T) {
	testEnvironment.Test(t, dataIngest(t))
}

func TestLabelVersionDetection(t *testing.T) {
	testEnvironment.Test(t, labelVersionDetection(t))
}

func TestReadOnlyCSIVolume(t *testing.T) {
	testEnvironment.Test(t, readOnlyCSIVolume(t))
}

func TestAppOnlyWithoutCSI(t *testing.T) {
	testEnvironment.Test(t, appOnlyWithoutCSI(t))
}
