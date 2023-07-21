//go:build e2e

package edgeconnect

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	ecComponentName = "edgeconnect"
)

func Install(t *testing.T) features.Feature {
	startTimestamp := time.Now()
	testEdgeConnect := edgeconnect.NewBuilder().
		Build()

	secretConfig := tenant.GetSingleTenantSecret(t)

	builder := features.New(ecComponentName)

	// Register operator install
	operatorNamespaceBuilder := namespace.NewBuilder(testEdgeConnect.Namespace)

	useCsi := false
	assess.InstallOperatorFromSourceWithCustomNamespace(builder, operatorNamespaceBuilder.Build(), useCsi)
	assess.InstallEdgeConnect(builder, &secretConfig, testEdgeConnect)

	builder.Assess("status update", edgeconnect.WaitForTimestampUpdate(testEdgeConnect, startTimestamp))

	teardown.UninstallOperatorWithEdgeConnectFromSource(builder, useCsi)

	return builder.Feature()
}
