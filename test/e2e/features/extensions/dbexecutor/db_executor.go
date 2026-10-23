// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package dbexecutor

import (
	"testing"

	dynakubeapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	k8sobject "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("extensions-db-executor-rollout")
	testDatabaseID := "mysql"

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithExtensionsEECImageRef(t, componentDynakube.GetLatestEECImageTagURI(t)),
		componentDynakube.WithExtensionsDatabases(extensions.DatabaseSpec{ID: testDatabaseID + "-a"}, extensions.DatabaseSpec{ID: testDatabaseID + "-b"}, extensions.DatabaseSpec{ID: testDatabaseID + "-c"}),
		componentDynakube.WithExtensionsDBExecutorImageRef(t, componentDynakube.GetLatestDBExecutorImageTagURI(t)),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := componentDynakube.New(options...)

	componentDynakube.Install(builder, secretConfig, testDynakube)

	builder.Assess("active gate pod is running", activegate.CheckContainer(testDynakube))

	builder.Assess("extensions execution controller started", k8sstatefulset.IsReady(testDynakube.Extensions().GetExecutionControllerStatefulsetName(), testDynakube.Namespace))

	builder.Assess("extensions db-a datasource deployment started", k8sdeployment.IsReady(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID+"-a"), testDynakube.Namespace))
	builder.Assess("extensions db-b datasource deployment started", k8sdeployment.IsReady(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID+"-b"), testDynakube.Namespace))
	builder.Assess("extensions db-c datasource deployment started", k8sdeployment.IsReady(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID+"-c"), testDynakube.Namespace))

	builder.Assess("sql resolved image", k8sobject.Expect(testDynakube.Name, testDynakube.Namespace, func(dk *dynakubeapi.DynaKube) bool {
		return dk.Status.SQLExtensionExecutor.ResolvedImage == dk.Spec.Templates.SQLExtensionExecutor.ImageRef.String()
	}))

	return builder.Feature()
}
