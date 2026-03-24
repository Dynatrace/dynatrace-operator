package activegate

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
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
	builder := features.New("activegate-with-hpa")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL))

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	activeGateSSName := activegate.GetActiveGateStateFulSetName(&testDynakube, "activegate")
	builder.Assess("check if AG doesn't have any replica count set", dynakubeComponents.WaitForAGReplicas(&testDynakube, nil))
	builder.Assess("check if the AG statefulset has replicas set to 1", k8sstatefulset.WaitForReplicas(activeGateSSName, testDynakube.Namespace, 1))

	testHPA := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-autoscaler",
			Namespace: testDynakube.Namespace,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "StatefulSet",
				Name:       activeGateSSName,
				APIVersion: "apps/v1",
			},
			MinReplicas: hpaScaleReplicas,
			MaxReplicas: *hpaScaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if AG doesn't have any replica count set", dynakubeComponents.WaitForAGReplicas(&testDynakube, nil))
	builder.Assess("check if the AG statefulset has replicas autoscaled to 3", k8sstatefulset.WaitForReplicas(activeGateSSName, testDynakube.Namespace, *hpaScaleReplicas))

	builder.Teardown(k8shpa.Delete(testHPA))
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}

func WithHPAEnforceReplicas(t *testing.T) features.Feature {
	builder := features.New("activegate-with-hpa-enforce-replicas")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithActiveGateReplicas(hpaBaseReplicas))

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	activeGateSSName := activegate.GetActiveGateStateFulSetName(&testDynakube, "activegate")

	builder.Assess("check if AG has replica count set to 2", dynakubeComponents.WaitForAGReplicas(&testDynakube, hpaBaseReplicas))
	builder.Assess("check if the AG statefulset has replicas set to 2", k8sstatefulset.WaitForReplicas(activeGateSSName, testDynakube.Namespace, *hpaBaseReplicas))

	testHPA := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-autoscaler",
			Namespace: testDynakube.Namespace,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "StatefulSet",
				Name:       activeGateSSName,
				APIVersion: "apps/v1",
			},
			MinReplicas: hpaScaleReplicas,
			MaxReplicas: *hpaScaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if HPA updated the replica count", k8shpa.WaitForCurrentReplicas(testHPA, *hpaScaleReplicas))
	builder.Assess("check if AG still has replicas set to 2", dynakubeComponents.WaitForAGReplicas(&testDynakube, hpaBaseReplicas))
	builder.Assess("check if the AG statefulset replica count is enforced to 2", k8sstatefulset.WaitForReplicas(activeGateSSName, testDynakube.Namespace, *hpaBaseReplicas))

	testDynakube.Spec.ActiveGate.Replicas = nil
	dynakubeComponents.Update(builder, helpers.LevelAssess, testDynakube)
	builder.Assess("check if AG has no replicas set", dynakubeComponents.WaitForAGReplicas(&testDynakube, nil))
	builder.Assess("check if the AG statefulset was autoscaled to 3", k8sstatefulset.WaitForReplicas(activeGateSSName, testDynakube.Namespace, *hpaScaleReplicas))

	builder.Teardown(k8shpa.Delete(testHPA))
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}
