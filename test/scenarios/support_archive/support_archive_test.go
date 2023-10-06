//go:build e2e

package support_archive

import (
	"github.com/Dynatrace/dynatrace-operator/src/util/logger"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	log.SetLogger(logger.Factory.GetLogger("e2e-support-archive"))

	testEnvironment = environment.GetStandardKubeClusterEnvironment()
	testEnvironment.Run(m)
}

func TestSupportArchive(t *testing.T) {
	testEnvironment.Test(t, supportArchiveExecution(t))
}
