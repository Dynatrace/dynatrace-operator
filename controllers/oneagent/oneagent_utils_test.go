package oneagent

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace = "test-namespace"
)

func TestBuildLabels(t *testing.T) {
	l := buildLabels("my-name", "classic")
	assert.Equal(t, l["dynatrace.com/component"], "operator")
	assert.Equal(t, l["operator.dynatrace.com/instance"], "my-name")
	assert.Equal(t, l["operator.dynatrace.com/feature"], "classic")
}

func TestGetPodReadyState(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{},
		}}
	assert.True(t, getPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}}
	assert.True(t, getPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: false}}
	assert.False(t, getPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}, {Ready: true}}
	assert.True(t, getPodReadyState(pod))

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}, {Ready: false}}
	assert.False(t, getPodReadyState(pod))
}

func TestOneAgent_Validate(t *testing.T) {
	oa := newOneAgent()
	assert.Error(t, validate(oa))
	oa.Spec.APIURL = "https://f.q.d.n/api"
	assert.NoError(t, validate(oa))
}

func TestMigrationForDaemonSetWithoutAnnotation(t *testing.T) {
	oaKey := metav1.ObjectMeta{Name: "my-oneagent", Namespace: "my-namespace"}

	ds1 := &appsv1.DaemonSet{ObjectMeta: oaKey}

	ds2, err := newDaemonSetForCR(consoleLogger, &dynatracev1alpha1.DynaKube{ObjectMeta: oaKey}, &dynatracev1alpha1.FullStackSpec{}, "classic", "cluster1")
	assert.NoError(t, err)
	assert.NotEmpty(t, ds2.Annotations[activegate.AnnotationTemplateHash])

	assert.True(t, hasDaemonSetChanged(ds1, ds2))
}

func TestHasSpecChanged(t *testing.T) {
	runTest := func(msg string, exp bool, mod func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube)) {
		t.Run(msg, func(t *testing.T) {
			key := metav1.ObjectMeta{Name: "my-oneagent", Namespace: "my-namespace"}
			oldInstance := dynatracev1alpha1.DynaKube{ObjectMeta: key}
			newInstance := dynatracev1alpha1.DynaKube{ObjectMeta: key}

			mod(&oldInstance, &newInstance)

			ds1, err := newDaemonSetForCR(consoleLogger, &oldInstance, &oldInstance.Spec.ClassicFullStack, "classic", "cluster1")
			assert.NoError(t, err)

			ds2, err := newDaemonSetForCR(consoleLogger, &newInstance, &newInstance.Spec.ClassicFullStack, "classic", "cluster1")
			assert.NoError(t, err)

			assert.NotEmpty(t, ds1.Annotations[activegate.AnnotationTemplateHash])
			assert.NotEmpty(t, ds2.Annotations[activegate.AnnotationTemplateHash])

			assert.Equal(t, exp, hasDaemonSetChanged(ds1, ds2))
		})
	}

	runTest("no changes", false, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {})

	runTest("image added", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		new.Spec.OneAgent.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image set but no change", false, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.OneAgent.Image = "docker.io/dynatrace/oneagent"
		new.Spec.OneAgent.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.OneAgent.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("image changed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.OneAgent.Image = "registry.access.redhat.com/dynatrace/oneagent"
		new.Spec.OneAgent.Image = "docker.io/dynatrace/oneagent"
	})

	runTest("argument removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.Args = []string{"INFRA_ONLY=1", "--set-host-property=OperatorVersion=snapshot"}
		new.Spec.ClassicFullStack.Args = []string{"INFRA_ONLY=1"}
	})

	runTest("argument changed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.Args = []string{"INFRA_ONLY=1"}
		new.Spec.ClassicFullStack.Args = []string{"INFRA_ONLY=0"}
	})

	runTest("all arguments removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.Args = []string{"INFRA_ONLY=1"}
	})

	runTest("resources added", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		new.Spec.ClassicFullStack.Resources = newResourceRequirements()
	})

	runTest("resources removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.Resources = newResourceRequirements()
	})

	runTest("resources removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.Resources = newResourceRequirements()
	})

	runTest("priority class added", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		new.Spec.ClassicFullStack.PriorityClassName = "class"
	})

	runTest("priority class removed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.PriorityClassName = "class"
	})

	runTest("priority class set but no change", false, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.PriorityClassName = "class"
		new.Spec.ClassicFullStack.PriorityClassName = "class"
	})

	runTest("priority class changed", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		old.Spec.ClassicFullStack.PriorityClassName = "some class"
		new.Spec.ClassicFullStack.PriorityClassName = "other class"
	})

	runTest("dns policy added", true, func(old *dynatracev1alpha1.DynaKube, new *dynatracev1alpha1.DynaKube) {
		new.Spec.ClassicFullStack.DNSPolicy = corev1.DNSClusterFirst
	})
}

func TestWaitPodReadyState(t *testing.T) {
	t.Run(`waitPodReadyState waits for pod to be ready`, func(t *testing.T) {
		labels := map[string]string{
			testKey: testValue,
		}
		pod1 := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels:    labels,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			}}
		pod2 := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + "-2",
				Namespace: testNamespace,
				Labels:    labels,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			}}

		waitSecs := uint16(1)
		//clt := fake.NewFakeClientWithScheme(scheme.Scheme, &pod1, &pod2)
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&pod1, &pod2).
			Build()

		r := &ReconcileOneAgent{
			client: clt,
		}
		err := r.waitPodReadyState(pod1, labels, waitSecs)
		assert.NoError(t, err)
	})
	t.Run(`waitPodReadyState returns error if pod does not become ready`, func(t *testing.T) {
		labels := map[string]string{
			testKey: testValue,
		}
		pod1 := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels:    labels,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			}}
		waitSecs := uint16(1)

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&pod1).
			Build()

		r := &ReconcileOneAgent{
			client: clt,
		}
		err := r.waitPodReadyState(pod1, labels, waitSecs)
		assert.Error(t, err)
	})
}

func newOneAgent() *dynatracev1alpha1.DynaKube {
	return &dynatracev1alpha1.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DynaKube",
			APIVersion: "dynatrace.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-oneagent",
			Namespace: "my-namespace",
			UID:       "69e98f18-805a-42de-84b5-3eae66534f75",
		},
	}
}

func newResourceRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    parseQuantity("10m"),
			"memory": parseQuantity("100Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    parseQuantity("20m"),
			"memory": parseQuantity("200Mi"),
		},
	}
}

func parseQuantity(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}
