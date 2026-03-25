//go:build e2e

package telemetryingest

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8shpa"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/require"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	scaleReplicas = ptr.To(int32(3))
	baseReplicas  = ptr.To(int32(2))
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
			MinReplicas: scaleReplicas,
			MaxReplicas: *scaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if the otelc statefulset has replicas set to 3", k8sstatefulset.WaitForReplicas(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace, *scaleReplicas))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.Teardown(k8shpa.Delete(testHPA))
	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}

func EnforceReplicas(t *testing.T) features.Feature {
	builder := features.New("telemetryingest-enforce-replicas")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithTelemetryIngestEnabled(true),
		componentDynakube.WithOTelCollectorImageRef(),
		componentDynakube.WithOTelCollectorReplicas(baseReplicas),
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("check if the otelc statefulset has replicas set to 2", k8sstatefulset.WaitForReplicas(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace, *baseReplicas))

	builder.Assess("scale explicitly otelc statefulset replicas to 3", func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		ss, err := k8sstatefulset.Get(ctx, resources, testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace)
		require.NoError(t, err)
		ss.Spec.Replicas = scaleReplicas
		require.NoError(t, k8sstatefulset.Update(ctx, resources, &ss))

		return ctx
	})

	builder.Assess("check if otelc replicas were rolled back to 2", k8sstatefulset.WaitForReplicas(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace, *baseReplicas))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}
