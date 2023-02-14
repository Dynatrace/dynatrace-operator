package autoscaler

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace = "imaginary"
)

var (
	testDynaKube = &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "anonym-dt-kube",
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureSyntheticLocationEntityId: "doctored",
			},
		},
	}

	testStatefulSet = &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "anonym-sts",
			Labels: map[string]string{
				"some": "another",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
	}
)

func TestToCreateSyntheticAutoscaler(t *testing.T) {
	assertion := assert.New(t)
	reconciler := newReconciler(fake.NewClient(), testDynaKube)
	toAssertCreated := func(t *testing.T) {
		require.NoError(t, reconciler.Reconcile())
	}
	t.Run("for-errorless", toAssertCreated)

	autoscalerName := testDynaKube.Name + "-" + SynAutoscaler
	toAssertName := func(t *testing.T) {
		autoscalers, err := reconciler.autoscalers.List(
			client.InNamespace(testNamespace))
		require.NoError(t, err)
		assertion.Len(autoscalers.Items, 1)
		assertion.Equal(
			autoscalers.Items[0].Name,
			autoscalerName,
			"auto-scaler name: %s",
			autoscalerName)
	}
	t.Run("by-name", toAssertName)
}

func newReconciler(
	client client.Client,
	dynaKube *dynatracev1beta1.DynaKube,
) *Reconciler {
	reconciler := NewReconciler(
		context.TODO(),
		client,
		client,
		scheme.Scheme,
		dynaKube,
		testStatefulSet,
	)

	return reconciler.(*Reconciler)
}

func TestToUpdateSyntheticAutoscaler(t *testing.T) {
	assertion := assert.New(t)
	reconciler := newReconciler(fake.NewClient(), testDynaKube)

	autoscaler, err := reconciler.builder.newAutoscaler()
	toAssertBuild := func(t *testing.T) {
		require.NoError(t, err)
	}
	t.Run("for-errorless-build", toAssertBuild)

	toUpdateDynaKube := testDynaKube.DeepCopy()
	autoscalerMaxReplicas := int32(11)
	toUpdateDynaKube.ObjectMeta.Annotations[dynatracev1beta1.AnnotationFeatureSyntheticAutoscalerMaxReplicas] = fmt.Sprint(autoscalerMaxReplicas)

	reconciler.builder.DynaKube = toUpdateDynaKube
	k8sRequests := fake.NewClient(autoscaler)
	reconciler.autoscalers.Reader = k8sRequests
	reconciler.autoscalers.Client = k8sRequests
	toAssertUpdated := func(t *testing.T) {
		require.NoError(t, reconciler.Reconcile())
	}
	t.Run("for-errorless-update", toAssertUpdated)

	toAssertMaxReplicas := func(t *testing.T) {
		autoscalers, err := reconciler.autoscalers.List(
			client.InNamespace(testNamespace))
		require.NoError(t, err)
		assertion.Len(autoscalers.Items, 1)
		assertion.Equal(
			autoscalers.Items[0].Spec.MaxReplicas,
			autoscalerMaxReplicas,
			"auto-scaler max replicas: %s",
			autoscalerMaxReplicas)
	}
	t.Run("by-max-replicas", toAssertMaxReplicas)
}

func TestToIgnoreSyntheticAutoscaler(t *testing.T) {
	assertion := assert.New(t)
	reconciler := newReconciler(fake.NewClient(), testDynaKube)

	autoscaler, err := reconciler.builder.newAutoscaler()
	toAssertBuild := func(t *testing.T) {
		require.NoError(t, err)
	}
	t.Run("for-errorless-build", toAssertBuild)

	k8sRequests := fake.NewClient(autoscaler)
	reconciler.autoscalers.Reader = k8sRequests
	reconciler.autoscalers.Client = k8sRequests
	reconciler.foundAutoscaler = autoscaler
	toAssertIgnored := func(t *testing.T) {
		assertion.True(
			reconciler.ignores(autoscaler),
			"auto-scaler hash equality")
	}
	t.Run("for-ignored", toAssertIgnored)
}

func TestToDeleteSyntheticAutoscaler(t *testing.T) {
	assertion := assert.New(t)
	reconciler := newReconciler(fake.NewClient(), testDynaKube)

	autoscaler, err := reconciler.builder.newAutoscaler()
	toAssertBuild := func(t *testing.T) {
		require.NoError(t, err)
	}
	t.Run("for-errorless-build", toAssertBuild)

	toDeleteDynaKube := testDynaKube.DeepCopy()
	delete(
		toDeleteDynaKube.ObjectMeta.Annotations,
		dynatracev1beta1.AnnotationFeatureSyntheticLocationEntityId)
	reconciler.builder.DynaKube = toDeleteDynaKube
	k8sRequests := fake.NewClient(autoscaler)
	reconciler.autoscalers.Reader = k8sRequests
	reconciler.autoscalers.Client = k8sRequests
	toAssertDeleted := func(t *testing.T) {
		require.NoError(t, reconciler.Reconcile())
		autoscalers, err := reconciler.autoscalers.List(
			client.InNamespace(testNamespace))
		require.NoError(t, err)
		assertion.Len(autoscalers.Items, 0)
	}
	t.Run("for-errorless-delete", toAssertDeleted)
}
