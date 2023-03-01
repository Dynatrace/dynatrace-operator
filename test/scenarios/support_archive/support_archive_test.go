//go:build e2e

package support_archive

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

func TestSupportArchive(t *testing.T) {
	testEnvironment.Test(t, supportArchiveExecution(t))
}
