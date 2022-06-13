package pod_mutator

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

type podMutatorEventRecorder struct {
	dynakube *dynatracev1beta1.DynaKube
	pod      *corev1.Pod
	recorder record.EventRecorder
}

func newPodMutatorEventRecorder(recorder record.EventRecorder) podMutatorEventRecorder {
	return podMutatorEventRecorder{recorder: recorder}
}

func (event *podMutatorEventRecorder) sendPodInjectEvent() {
	event.recorder.Eventf(event.dynakube,
		corev1.EventTypeNormal,
		injectEvent,
		"Injecting the necessary info into pod %s in namespace %s", event.pod.GenerateName, event.pod.Namespace)
}

func (event *podMutatorEventRecorder) sendPodUpdateEvent() {
	event.recorder.Eventf(event.dynakube,
		corev1.EventTypeNormal,
		updatePodEvent,
		"Updating pod %s in namespace %s with missing containers", event.pod.GenerateName, event.pod.Namespace)
}

func (event *podMutatorEventRecorder) sendMissingDynaKubeEvent(namespaceName, dynakubeName string) {
	template := "Namespace '%s' is assigned to DynaKube instance '%s' but this instance doesn't exist"
	event.recorder.Eventf(
		&dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dynakubeName, Namespace: namespaceName}},
		corev1.EventTypeWarning,
		missingDynakubeEvent,
		template, namespaceName, dynakubeName)
}
