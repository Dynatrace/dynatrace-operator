//go:build e2e

package dbexecutor

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/test/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("extensions-db-executor-rollout")
	testDatabaseID := "mysql"

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithExtensionsEECImageRefSpec(consts.EecImageRepo, consts.EecImageTag),
		componentDynakube.WithExtensionsDatabases(extensions.DatabaseSpec{ID: testDatabaseID + "-a"}, extensions.DatabaseSpec{ID: testDatabaseID + "-b"}, extensions.DatabaseSpec{ID: testDatabaseID + "-c"}),
		componentDynakube.WithExtensionsDBExecutorImageRefSpec(consts.DBExecutorImageRepo, consts.DBExecutorImageTag),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("active gate pod is running", activegate.CheckContainer(&testDynakube))

	builder.Assess("extensions execution controller started", k8sstatefulset.IsReady(testDynakube.Extensions().GetExecutionControllerStatefulsetName(), testDynakube.Namespace))

	builder.Assess("extensions db-a datasource deployment started", k8sdeployment.IsReady(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID+"-a"), testDynakube.Namespace))
	builder.Assess("extensions db-b datasource deployment started", k8sdeployment.IsReady(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID+"-b"), testDynakube.Namespace))
	builder.Assess("extensions db-c datasource deployment started", k8sdeployment.IsReady(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID+"-c"), testDynakube.Namespace))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}
