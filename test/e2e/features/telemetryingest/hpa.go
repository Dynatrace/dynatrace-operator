package telemetryingest

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8shpa"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
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
	builder := features.New("telemetryingest-with-hpa")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithTelemetryIngestEnabled(true),
		componentDynakube.WithOTelCollectorImageRef(),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("check if otelc doesn't have any replica count set", componentDynakube.WaitForOtelCollectorReplicas(&testDynakube, nil))
	builder.Assess("check if the otelc statefulset has replicas set to 1", k8sstatefulset.WaitForReplicas(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace, 1))

	testHPA := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-autoscaler",
			Namespace: testDynakube.Namespace,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "StatefulSet",
				Name:       testDynakube.OtelCollectorStatefulsetName(),
				APIVersion: "apps/v1",
			},
			MinReplicas: hpaScaleReplicas,
			MaxReplicas: *hpaScaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if otelc doesn't have any replica count set", componentDynakube.WaitForOtelCollectorReplicas(&testDynakube, nil))
	builder.Assess("check if the otelc statefulset has replicas set to 3", k8sstatefulset.WaitForReplicas(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace, *hpaScaleReplicas))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.Teardown(k8shpa.Delete(testHPA))
	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}

func WithHPAEnforceReplicas(t *testing.T) features.Feature {
	builder := features.New("telemetryingest-with-hpa-enforce-replicas")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithTelemetryIngestEnabled(true),
		componentDynakube.WithOTelCollectorImageRef(),
		componentDynakube.WithOTelCollectorReplicas(ptr.To(int32(2))),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("check if otelc has replicas count set to 2", componentDynakube.WaitForOtelCollectorReplicas(&testDynakube, hpaBaseReplicas))
	builder.Assess("check if the otelc statefulset has replicas set to 2", k8sstatefulset.WaitForReplicas(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace, *hpaBaseReplicas))

	testHPA := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-autoscaler",
			Namespace: testDynakube.Namespace,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "StatefulSet",
				Name:       testDynakube.OtelCollectorStatefulsetName(),
				APIVersion: "apps/v1",
			},
			MinReplicas: hpaScaleReplicas,
			MaxReplicas: *hpaScaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if HPA updated the replica count", k8shpa.WaitForCurrentReplicas(testHPA, *hpaScaleReplicas))
	builder.Assess("check if otelc still has replicas set to 2", componentDynakube.WaitForOtelCollectorReplicas(&testDynakube, hpaBaseReplicas))
	builder.Assess("check if the otelc statefulset replica count 2 is enforced", k8sstatefulset.WaitForReplicas(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace, *hpaBaseReplicas))

	testDynakube.Spec.Templates.OpenTelemetryCollector.Replicas = nil
	componentDynakube.Update(builder, helpers.LevelAssess, testDynakube)
	builder.Assess("check if otelc has no replicas set", componentDynakube.WaitForOtelCollectorReplicas(&testDynakube, nil))
	builder.Assess("check if the otelc statefulset was autoscaled to 3", k8sstatefulset.WaitForReplicas(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace, *hpaScaleReplicas))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.Teardown(k8shpa.Delete(testHPA))
	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}
