package pod

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

type eventRecorder struct {
	dk       *dynakube.DynaKube
	pod      *corev1.Pod
	recorder record.EventRecorder
}

func newPodMutatorEventRecorder(recorder record.EventRecorder) eventRecorder {
	return eventRecorder{recorder: recorder}
}

func (er *eventRecorder) sendPodInjectEvent() {
	er.recorder.Eventf(er.dk,
		corev1.EventTypeNormal,
		injectEvent,
		"Injecting the necessary info into pod %s in namespace %s", er.pod.GenerateName, er.pod.Namespace)
}

func (er *eventRecorder) sendPodUpdateEvent() {
	er.recorder.Eventf(er.dk,
		corev1.EventTypeNormal,
		updatePodEvent,
		"Updating pod %s in namespace %s with missing containers", er.pod.GenerateName, er.pod.Namespace)
}

func (er *eventRecorder) sendMissingDynaKubeEvent(namespaceName, dynakubeName string) {
	template := "Namespace '%s' is assigned to DynaKube instance '%s' but this instance doesn't exist"
	er.recorder.Eventf(
		&dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: dynakubeName, Namespace: namespaceName}},
		corev1.EventTypeWarning,
		missingDynakubeEvent,
		template, namespaceName, dynakubeName)
}

func (er *eventRecorder) sendOneAgentAPMWarningEvent(webhookPod *corev1.Pod) {
	er.recorder.Event(webhookPod,
		corev1.EventTypeWarning,
		IncompatibleCRDEvent,
		"Unsupported OneAgentAPM CRD still present in cluster, please remove to proceed")
}
