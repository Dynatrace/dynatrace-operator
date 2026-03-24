package dbexecutor

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8shpa"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	hpaScaleReplicas = ptr.To(int32(3))
	hpaBaseReplicas  = ptr.To(int32(2))
)

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
	builder.Assess("check if the deployment has replicas set to 1", k8sdeployment.WaitForReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, 1))

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
			MinReplicas: hpaScaleReplicas,
			MaxReplicas: *hpaScaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if executor doesn't have any replica count set", componentDynakube.WaitForDBExecutorReplicas(&testDynakube, testDatabaseID, nil))
	builder.Assess("check if the deployment has replicas set to 3", k8sdeployment.WaitForReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, *hpaScaleReplicas))

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
		componentDynakube.WithExtensionsDatabases(extensions.DatabaseSpec{ID: testDatabaseID, Replicas: hpaBaseReplicas}),
		componentDynakube.WithExtensionsDBExecutorImageRef(),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("check if executor has replicas count set to 2", componentDynakube.WaitForDBExecutorReplicas(&testDynakube, testDatabaseID, hpaBaseReplicas))
	builder.Assess("check if the executor deployment has replicas set to 2", k8sdeployment.WaitForReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, *hpaBaseReplicas))

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
			MinReplicas: hpaScaleReplicas,
			MaxReplicas: *hpaScaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if HPA updated the replica count", k8shpa.WaitForCurrentReplicas(testHPA, *hpaScaleReplicas))
	builder.Assess("check if executor still has replicas set to 2", componentDynakube.WaitForDBExecutorReplicas(&testDynakube, testDatabaseID, hpaBaseReplicas))
	builder.Assess("check if the executor deployment replica count 2 is enforced", k8sdeployment.WaitForReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, *hpaBaseReplicas))

	testDynakube.Spec.Extensions.Databases[0].Replicas = nil
	componentDynakube.Update(builder, helpers.LevelAssess, testDynakube)
	builder.Assess("check if executor has no replicas set", componentDynakube.WaitForDBExecutorReplicas(&testDynakube, testDatabaseID, nil))
	builder.Assess("check if the executor deployment was autoscaled to 3", k8sdeployment.WaitForReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, *hpaScaleReplicas))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.Teardown(k8shpa.Delete(testHPA))
	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}
