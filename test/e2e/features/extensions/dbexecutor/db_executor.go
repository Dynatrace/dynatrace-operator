//go:build e2e

package dbexecutor

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8shpa"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/require"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	"k8s.io/utils/ptr"

	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("extensions-db-executor-rollout")
	testDatabaseID := "mysql"

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithExtensionsEECImageRef(),
		componentDynakube.WithExtensionsDatabases(extensions.DatabaseSpec{ID: testDatabaseID + "-a"}, extensions.DatabaseSpec{ID: testDatabaseID + "-b"}, extensions.DatabaseSpec{ID: testDatabaseID + "-c"}),
		componentDynakube.WithExtensionsDBExecutorImageRef(),
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

func WithHPA(t *testing.T) features.Feature {
	builder := features.New("extensions-db-executor-with-hpa")
	testDatabaseID := "mysql"

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithExtensionsEECImageRef(),
		componentDynakube.WithExtensionsDatabases(extensions.DatabaseSpec{ID: testDatabaseID}),
		componentDynakube.WithExtensionsDBExecutorImageRef(),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("check if executor doesn't have any replica count set", componentDynakube.WaitForDBExecutorReplicas(&testDynakube, testDatabaseID, nil))
	builder.Assess("check if the deployment has replicas set to 1", k8sdeployment.WaitForSpecReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, 1))

	testHPA := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-autoscaler",
			Namespace: testDynakube.Namespace,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID),
				APIVersion: "apps/v1",
			},
			MinReplicas: ptr.To(int32(3)),
			MaxReplicas: int32(3),
		},
	}

	builder.Assess("create hpa with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if executor doesn't have any replica count set", componentDynakube.WaitForDBExecutorReplicas(&testDynakube, testDatabaseID, nil))
	builder.Assess("check if the deployment has replicas set to 3", k8sdeployment.WaitForSpecReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, 3))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.Teardown(k8shpa.Delete(testHPA))
	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()

}

func WithHPAEnforceReplicas(t *testing.T) features.Feature {
	builder := features.New("extensions-db-executor-with-hpa-enforce-replicas")
	testDatabaseID := "mysql"

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithExtensionsEECImageRef(),
		componentDynakube.WithExtensionsDatabases(extensions.DatabaseSpec{ID: testDatabaseID, Replicas: ptr.To(int32(2))}),
		componentDynakube.WithExtensionsDBExecutorImageRef(),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("check if executor has replicas count set to 2", componentDynakube.WaitForDBExecutorReplicas(&testDynakube, testDatabaseID, ptr.To(int32(2))))
	builder.Assess("check if the executor deployment has replicas set to 2", k8sdeployment.WaitForSpecReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, 2))

	testHPA := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-autoscaler",
			Namespace: testDynakube.Namespace,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID),
				APIVersion: "apps/v1",
			},
			MinReplicas: ptr.To(int32(3)),
			MaxReplicas: int32(3),
		},
	}

	builder.Assess("create hpa with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if HPA updated the replica count", k8shpa.WaitCurrentReplicas(testHPA, 3))
	builder.Assess("check if executor still has replicas set to 2", componentDynakube.WaitForDBExecutorReplicas(&testDynakube, testDatabaseID, ptr.To(int32(2))))
	builder.Assess("check if the executor deployment replica count 2 is enforced", k8sdeployment.WaitForSpecReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, 2))

	builder.Assess("remove enforced replicas", updateReplicas(&testDynakube, testDatabaseID, nil))
	builder.Assess("check if executor has no replicas set", componentDynakube.WaitForDBExecutorReplicas(&testDynakube, testDatabaseID, nil))
	builder.Assess("check if the executor deployment was autoscaled to 3", k8sdeployment.WaitForSpecReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, 3))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.Teardown(k8shpa.Delete(testHPA))
	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()

}

func updateReplicas(dk *dynakube.DynaKube, dbId string, replicas *int32) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Get(ctx, dk.Name, dk.Namespace, dk)
		require.NoError(t, err)

		dbs := dk.Spec.Extensions.Databases

		for i := range dbs {
			if dbs[i].ID == dbId {
				dbs[i].Replicas = replicas
				break
			}
		}

		err = envConfig.Client().Resources().Update(ctx, dk)
		require.NoError(t, err)

		return ctx
	}
}
