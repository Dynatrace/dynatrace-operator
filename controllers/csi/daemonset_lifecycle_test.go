package dtcsi

import (
	"context"
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/api/v1"
	"github.com/Dynatrace/dynatrace-operator/controllers"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	otherTestDynakube = "other-test-dynakube"
)

func Test_ConfigureCSIDriver_Enable(t *testing.T) {
	dynakube := prepareDynakube(testDynakube)
	fakeClient := prepareFakeClient()
	dkState := prepareDynakubeState(dynakube, true)

	err := ConfigureCSIDriver(fakeClient, scheme.Scheme, testOperatorPodName, testNamespace, dkState, 10)
	require.NoError(t, err)

	csiDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      DaemonSetName,
		Namespace: testNamespace,
	}, csiDaemonSet)
	require.NoError(t, err)

	assert.NotNil(t, csiDaemonSet.OwnerReferences)
	assert.Len(t, csiDaemonSet.OwnerReferences, 1)
	assert.Contains(t, csiDaemonSet.OwnerReferences, getOwnerReferenceFromDynakube(dynakube))
}

func Test_ConfigureCSIDriver_Disable(t *testing.T) {
	dynakube := prepareDynakube(testDynakube)
	fakeClient := prepareFakeClientWithEnabledCSI(dynakube)
	dkState := prepareDynakubeState(dynakube, false)

	err := ConfigureCSIDriver(fakeClient, scheme.Scheme, testOperatorPodName, testNamespace, dkState, 10)
	require.NoError(t, err)

	updatedDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      DaemonSetName,
		Namespace: testNamespace,
	}, updatedDaemonSet)
	require.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
}

func Test_ConfigureCSIDriver_RemoveDynakube_CSIStaysDisabled(t *testing.T) {
	dynakube := prepareDynakube(testDynakube)
	otherDynakube := prepareDynakube(otherTestDynakube)
	fakeClient := prepareFakeClientWithEnabledCSI(dynakube, otherDynakube)
	dkState := prepareDynakubeState(dynakube, false)

	err := ConfigureCSIDriver(fakeClient, scheme.Scheme, testOperatorPodName, testNamespace, dkState, 10)
	require.NoError(t, err)

	updatedDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      DaemonSetName,
		Namespace: testNamespace,
	}, updatedDaemonSet)
	require.NoError(t, err)

	require.NotNil(t, updatedDaemonSet.OwnerReferences)
	assert.Len(t, updatedDaemonSet.OwnerReferences, 1)
	assert.Contains(t, updatedDaemonSet.OwnerReferences, getOwnerReferenceFromDynakube(otherDynakube))
	assert.NotContains(t, updatedDaemonSet.OwnerReferences, getOwnerReferenceFromDynakube(dynakube))
}

func Test_ConfigureCSIDriver_AddDynakube_CSIStaysEnabled(t *testing.T) {
	dynakube := prepareDynakube(testDynakube)
	otherDynakube := prepareDynakube(otherTestDynakube)
	fakeClient := prepareFakeClientWithEnabledCSI(otherDynakube)
	dkState := prepareDynakubeState(dynakube, true)

	err := ConfigureCSIDriver(fakeClient, scheme.Scheme, testOperatorPodName, testNamespace, dkState, 10)
	require.NoError(t, err)

	updatedDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      DaemonSetName,
		Namespace: testNamespace,
	}, updatedDaemonSet)
	require.NoError(t, err)

	require.NotNil(t, updatedDaemonSet.OwnerReferences)
	assert.Len(t, updatedDaemonSet.OwnerReferences, 2)
	assert.Contains(t, updatedDaemonSet.OwnerReferences, getOwnerReferenceFromDynakube(otherDynakube))
	assert.Contains(t, updatedDaemonSet.OwnerReferences, getOwnerReferenceFromDynakube(dynakube))
}

func getOwnerReferenceFromDynakube(dynakube *dynatracev1.DynaKube) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         dynakube.APIVersion,
		Kind:               dynakube.Kind,
		Name:               dynakube.Name,
		UID:                dynakube.UID,
		Controller:         pointer.BoolPtr(false),
		BlockOwnerDeletion: pointer.BoolPtr(false),
	}
}

func prepareFakeClientWithEnabledCSI(dynakubes ...*dynatracev1.DynaKube) client.Client {
	var ownerReferences []metav1.OwnerReference
	for _, dynakube := range dynakubes {
		ownerReferences = append(ownerReferences, getOwnerReferenceFromDynakube(dynakube))
	}

	csiDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       testNamespace,
			Name:            DaemonSetName,
			OwnerReferences: ownerReferences,
		},
	}

	fakeClient := prepareFakeClient(csiDaemonSet)
	return fakeClient
}

func prepareDynakubeState(dynakube *dynatracev1.DynaKube, enableCodeModules bool) *controllers.DynakubeState {
	log := logger.NewDTLogger()

	if enableCodeModules {
		dynakube.Spec = dynatracev1.DynaKubeSpec{
			OneAgent: dynatracev1.OneAgentSpec{
				ApplicationMonitoring: &dynatracev1.ApplicationMonitoringSpec{},
			},
		}
	}

	return &controllers.DynakubeState{
		Log:      log,
		Instance: dynakube,
	}
}
