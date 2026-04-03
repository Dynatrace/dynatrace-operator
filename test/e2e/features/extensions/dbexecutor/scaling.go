//go:build e2e

package dbexecutor

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8shpa"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	scaleReplicas = ptr.To(int32(3))
	baseReplicas  = ptr.To(int32(2))
)

func WithHPA(t *testing.T) features.Feature {
	builder := features.New("extensions-db-executor-with-hpa")
	testDatabaseID := "mysql"

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithExtensionsEECImageRef(t),
		componentDynakube.WithExtensionsDatabases(extensions.DatabaseSpec{ID: testDatabaseID}),
		componentDynakube.WithExtensionsDBExecutorImageRef(t),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

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
			MinReplicas: scaleReplicas,
			MaxReplicas: *scaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if the deployment has replicas set to 3", k8sdeployment.WaitForReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, *scaleReplicas))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.Teardown(k8shpa.Delete(testHPA))
	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}

func EnforceReplicas(t *testing.T) features.Feature {
	builder := features.New("extensions-db-executor-enforce-replicas")
	testDatabaseID := "mysql"

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithExtensionsEECImageRef(t),
		componentDynakube.WithExtensionsDatabases(extensions.DatabaseSpec{ID: testDatabaseID, Replicas: baseReplicas}),
		componentDynakube.WithExtensionsDBExecutorImageRef(t),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("scale executor deployment replicas to 3", k8sdeployment.Update(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, func(d *appsv1.Deployment) {
		d.Spec.Replicas = scaleReplicas
	}))

	builder.Assess("check if executor deployment replicas were rolled back to 2", k8sdeployment.WaitForReplicas(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID), testDynakube.Namespace, *baseReplicas))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}
