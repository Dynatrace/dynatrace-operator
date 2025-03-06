package pod

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testImage         = "test-image"
	testNamespaceName = "test-namespace"
	testClusterID     = "test-cluster-id"
	testPodName       = "test-pod"
	testDynakubeName  = "test-dynakube"
)

func getTestPodWithInjectionDisabled() *corev1.Pod {
	pod := getTestPod()
	pod.Annotations = map[string]string{
		dtwebhook.AnnotationDynatraceInject: "false",
	}

	return pod
}

func getTestPodWithOcDebugPodAnnotations() *corev1.Pod {
	pod := getTestPod()
	pod.Annotations = map[string]string{
		ocDebugAnnotationsContainer: "true",
		ocDebugAnnotationsResource:  "true",
	}

	return pod
}

func createTestWebhook(v1, v2 dtwebhook.PodInjector, objects []client.Object) *webhook {
	decoder := admission.NewDecoder(scheme.Scheme)

	return &webhook{
		v1: v1,
		v2: v2,
		apiReader:        fake.NewClient(objects...),
		decoder:          decoder,
		webhookNamespace: testNamespaceName,
	}
}

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: getCloudNativeSpec(),
		},
	}
}

func getTestDynakubeDefaultAppMon() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: getTestDynakubeMeta(),
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
			},
		},
	}
}

func getTestDynakubeMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      testDynakubeName,
		Namespace: testNamespaceName,
	}
}

func getCloudNativeSpec() oneagent.Spec {
	return oneagent.Spec{
		CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{},
		},
	}
}
