package oneagent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func mergeLabels(labels ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range labels {
		for k, v := range m {
			res[k] = v
		}
	}

	return res
}

// buildLabels returns generic labels based on the name given for a Dynatrace OneAgent
func buildLabels(name string, feature string) map[string]string {
	return map[string]string{
		"dynatrace.com/component":         "operator",
		"operator.dynatrace.com/instance": name,
		"operator.dynatrace.com/feature":  feature,
	}
}

// getPodReadyState determines the overall ready state of a Pod.
// Returns true if all containers in the Pod are ready.
func getPodReadyState(p *corev1.Pod) bool {
	ready := true
	for _, c := range p.Status.ContainerStatuses {
		ready = ready && c.Ready
	}

	return ready
}

// validate sanity checks if essential fields in the custom resource are available
//
// Return an error in the following conditions
// - APIURL empty
func validate(cr *dynatracev1alpha1.DynaKube) error {
	var msg []string
	if cr.Spec.APIURL == "" {
		msg = append(msg, ".spec.apiUrl is missing")
	}
	if len(msg) > 0 {
		return errors.New(strings.Join(msg, ", "))
	}
	return nil
}

func (r *ReconcileOneAgent) determineOneAgentPhase(instance *dynatracev1alpha1.DynaKube) (bool, error) {
	var phaseChanged bool
	dsActual := &appsv1.DaemonSet{}
	instanceName := fmt.Sprintf("%s-%s", instance.Name, r.feature)
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instanceName, Namespace: instance.Namespace}, dsActual)

	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		phaseChanged = instance.Status.Phase != dynatracev1alpha1.Error
		instance.Status.Phase = dynatracev1alpha1.Error
		return phaseChanged, err
	}

	if dsActual.Status.NumberReady == dsActual.Status.CurrentNumberScheduled {
		phaseChanged = instance.Status.Phase != dynatracev1alpha1.Running
		instance.Status.Phase = dynatracev1alpha1.Running
	} else {
		phaseChanged = instance.Status.Phase != dynatracev1alpha1.Deploying
		instance.Status.Phase = dynatracev1alpha1.Deploying
	}

	return phaseChanged, nil
}

func (r *ReconcileOneAgent) waitPodReadyState(pod corev1.Pod, labels map[string]string, waitSecs uint16) error {
	var status error

	listOps := []client.ListOption{
		client.InNamespace(pod.Namespace),
		client.MatchingLabels(labels),
	}

	for splay := uint16(0); splay < waitSecs; splay += splayTimeSeconds {
		time.Sleep(time.Duration(splayTimeSeconds) * time.Second)

		// The actual selector we need is,
		// "spec.nodeName=<pod.Spec.NodeName>,status.phase=Running,metadata.name!=<pod.Name>"
		//
		// However, the client falls back to a cached implementation for .List() after the first attempt, which
		// is not able to handle our query so the function fails. Because of this, we're getting all the pods and
		// filtering it ourselves.
		podList := &corev1.PodList{}
		status = r.client.List(context.TODO(), podList, listOps...)
		if status != nil {
			continue
		}

		var foundPods []*corev1.Pod
		for i := range podList.Items {
			p := &podList.Items[i]
			if p.Spec.NodeName != pod.Spec.NodeName || p.Status.Phase != corev1.PodRunning ||
				p.ObjectMeta.Name == pod.Name {
				continue
			}
			foundPods = append(foundPods, p)
		}

		if n := len(foundPods); n == 0 {
			status = fmt.Errorf("waiting for pod to be recreated on node: %s", pod.Spec.NodeName)
		} else if n == 1 && getPodReadyState(foundPods[0]) {
			break
		} else if n > 1 {
			status = fmt.Errorf("too many pods found: expected=1 actual=%d", n)
		}
	}

	return status
}
