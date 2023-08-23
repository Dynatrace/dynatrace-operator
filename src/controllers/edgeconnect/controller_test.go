package edgeconnect

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testName      = "test-name-edgeconnectv1alpha1"
	testNamespace = "test-namespace"
)

func TestIsError(t *testing.T) {
	assert.False(t, k8serrors.IsConflict(nil))
}

func TestReconcile(t *testing.T) {
	t.Run("Create works with minimal setup", func(t *testing.T) {
		controller := &Controller{
			client:    fake.NewClient(),
			apiReader: fake.NewClient(),
		}
		result, err := controller.Reconcile(context.TODO(), reconcile.Request{})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
	t.Run("Timestamp update in EdgeConnect status works", func(t *testing.T) {
		now := metav1.Now()
		instance := &edgeconnectv1alpha1.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnectv1alpha1.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
			},
			Status: edgeconnectv1alpha1.EdgeConnectStatus{
				UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
				Version: status.VersionStatus{
					LastProbeTimestamp: &now,
					ImageID:            "docker.io/dynatrace/edgeconnect:latest",
				},
			},
		}

		controller := createFakeClientAndReconciler(instance)
		controller.timeProvider.Freeze()

		result, err := controller.Reconcile(context.TODO(), reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testName},
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		err = controller.apiReader.Get(context.TODO(), client.ObjectKey{Name: instance.Name, Namespace: instance.Namespace}, instance)
		require.NoError(t, err)
		// Fake client drops seconds, so we have to do the same
		expectedTimestamp := controller.timeProvider.Now().Truncate(time.Second)
		assert.Equal(t, expectedTimestamp, instance.Status.UpdatedTimestamp.Time)
	})
}

func createFakeClientAndReconciler(instance *edgeconnectv1alpha1.EdgeConnect) *Controller {
	fakeClient := fake.NewClientWithIndex(instance)

	controller := &Controller{
		client:       fakeClient,
		apiReader:    fakeClient,
		scheme:       scheme.Scheme,
		timeProvider: timeprovider.New(),
	}
	return controller
}
