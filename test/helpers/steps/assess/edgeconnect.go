//go:build e2e

package assess

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	edgeconnectv1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func CreateEdgeConnect(builder *features.FeatureBuilder, secretConfig *tenant.EdgeConnectSecret, testEdgeConnect edgeconnectv1beta1.EdgeConnect) {
	if secretConfig != nil {
		builder.Assess("create edgeconnect client secret", tenant.CreateClientSecret(*secretConfig, fmt.Sprintf("%s-client-secret", testEdgeConnect.Name), testEdgeConnect.Namespace))
		builder.Assess("create edgeconnect docker pull secret", tenant.CreateDockerPullSecret(*secretConfig, fmt.Sprintf("%s-docker-pull-secret", testEdgeConnect.Name), testEdgeConnect.Namespace))
	}
	builder.Assess(
		fmt.Sprintf("'%s' edgeconnect created", testEdgeConnect.Name),
		edgeconnect.Create(testEdgeConnect))
}

func VerifyEdgeConnectStartup(builder *features.FeatureBuilder, testEdgeConnect edgeconnectv1beta1.EdgeConnect) {
	builder.Assess(
		fmt.Sprintf("'%s' edgeconnect phase changes to 'Running'", testEdgeConnect.Name),
		edgeconnect.WaitForEdgeConnectPhase(testEdgeConnect, status.Running))
}
