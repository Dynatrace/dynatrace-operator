//go:build e2e

package codemodules

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/proxy"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.Run(m)
}

func TestCloudNative(t *testing.T) {
	testEnvironment.Test(t, CodeModules(t, false))
}

func TestCloudNativeWithProxy(t *testing.T) {
	testEnvironment.BeforeEachTest(istio.AssertIstioNamespace())
	testEnvironment.BeforeEachTest(istio.AssertIstiodDeployment())
	testEnvironment.Test(t, withProxy(t, proxy.ProxySpec))
}

func TestCloudNativeWithProxyCustomCA(t *testing.T) {
	testEnvironment.BeforeEachTest(istio.AssertIstioNamespace())
	testEnvironment.BeforeEachTest(istio.AssertIstiodDeployment())
	testEnvironment.Test(t, withProxyCA(t, proxy.ProxySpec))
}
