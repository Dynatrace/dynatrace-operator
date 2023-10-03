//go:build e2e

package network_zones

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/environment"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	log.SetLogger(logger.Factory.GetLogger("e2e-cloudnative-network-zones"))
	testEnvironment = environment.GetStandardKubeClusterEnvironment()
	testEnvironment.Run(m)
}

func TestNetworkZones(t *testing.T) {
	testEnvironment.Test(t, networkZones(t))
}
