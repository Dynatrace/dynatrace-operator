//go:build e2e

package assess

import (
	"fmt"

	edgeconnectv1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallEdgeConnect(builder *features.FeatureBuilder, secretConfig *tenant.Secret, testEdgeConnect edgeconnectv1beta1.EdgeConnect) {
	CreateEdgeConnect(builder, secretConfig, testEdgeConnect)
}

func CreateEdgeConnect(builder *features.FeatureBuilder, secretConfig *tenant.Secret, testEdgeConnect edgeconnectv1beta1.EdgeConnect) {
	if secretConfig != nil {
		builder.Assess("created tenant secret", tenant.CreateTenantSecret(*secretConfig, testEdgeConnect.Name, testEdgeConnect.Namespace))
	}
	builder.Assess(
		fmt.Sprintf("'%s' edgeconnect created", testEdgeConnect.Name),
		edgeconnect.Create(testEdgeConnect))
}
