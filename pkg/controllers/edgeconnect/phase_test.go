package edgeconnect

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestEdgeConnectPhaseChanges(t *testing.T) {
	ec := &edgeconnect.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: edgeconnect.EdgeConnectSpec{},
	}

	t.Run("no edgeConnect deployment in cluster -> deploying", func(t *testing.T) {
		fakeClient := fake.NewClient()
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineEdgeConnectPhase(ec)
		assert.Equal(t, status.Deploying, phase)
	})

	t.Run("error accessing k8s api -> error", func(t *testing.T) {
		fakeClient := errorClient{}
		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineEdgeConnectPhase(ec)
		assert.Equal(t, status.Error, phase)
	})

	t.Run("edgeConnect pods not ready -> deploying", func(t *testing.T) {
		replicas, readyReplicas := int32(1), int32(0)
		objects := []client.Object{
			createDeployment(testNamespace, testName, replicas, readyReplicas),
		}

		fakeClient := fake.NewClient(objects...)

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineEdgeConnectPhase(ec)
		assert.Equal(t, status.Deploying, phase)
	})

	t.Run("edgeConnect deployed -> running", func(t *testing.T) {
		replicas, readyReplicas := int32(1), int32(1)
		objects := []client.Object{
			createDeployment(testNamespace, testName, replicas, readyReplicas),
		}

		fakeClient := fake.NewClient(objects...)

		controller := &Controller{
			client:    fakeClient,
			apiReader: fakeClient,
		}
		phase := controller.determineEdgeConnectPhase(ec)
		assert.Equal(t, status.Running, phase)
	})
}
