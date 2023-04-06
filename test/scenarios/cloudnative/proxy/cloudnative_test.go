//go:build e2e

package cloudnativeproxy

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/proxy"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.Run(m)
}

func TestCloudNativeWithProxy(t *testing.T) {
	testEnvironment.Test(t, withProxy(t, proxy.ProxySpec))
}
