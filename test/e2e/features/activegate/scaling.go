//go:build e2e

package activegate

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8shpa"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
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
	builder := features.New("activegate-with-hpa")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL))

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	activeGateSSName := activegate.GetActiveGateStateFulSetName(&testDynakube, "activegate")

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
			MinReplicas: scaleReplicas,
			MaxReplicas: *scaleReplicas,
		},
	}

	builder.Assess("create HPA with min replicas 3", k8shpa.Create(testHPA))
	builder.Assess("check if the AG statefulset has replicas autoscaled to 3", k8sstatefulset.WaitForReplicas(activeGateSSName, testDynakube.Namespace, *scaleReplicas))

	builder.Teardown(k8shpa.Delete(testHPA))
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}

func EnforceReplicas(t *testing.T) features.Feature {
	builder := features.New("activegate-enforce-replicas")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithActiveGateReplicas(baseReplicas))

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	activeGateSSName := activegate.GetActiveGateStateFulSetName(&testDynakube, "activegate")

	builder.Assess("scale AG statefulset replicas to 3", k8sstatefulset.Update(activeGateSSName, testDynakube.Namespace, func(ss *appsv1.StatefulSet) {
		ss.Spec.Replicas = scaleReplicas
	}))

	builder.Assess("check if the AG statefulset was rolled back to 2", k8sstatefulset.WaitForReplicas(activeGateSSName, testDynakube.Namespace, *baseReplicas))

	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}
