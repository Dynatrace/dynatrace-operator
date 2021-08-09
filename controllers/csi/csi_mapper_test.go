package dtcsi

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testDynakube = "test-dynakube"

	testOperatorImage = "test-operator-image"
)

func Test_ConfigureCSIDriver_Enable(t *testing.T) {
	fakeClient := prepareFakeClient()
	rec := prepareReconciliation(true)

	err := ConfigureCSIDriver(fakeClient, scheme.Scheme, testOperatorPodName, testNamespace, rec, 10)
	require.NoError(t, err)

	configMap := &corev1.ConfigMap{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      CsiMapperConfigMapName,
		Namespace: testNamespace,
	}, configMap)
	require.NoError(t, err)

	assert.NotNil(t, configMap.Data)
	assert.Len(t, configMap.Data, 1)
	assert.Contains(t, configMap.Data, testDynakube)
	assert.Equal(t, "", configMap.Data[testDynakube])

	csiDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      DaemonSetName,
		Namespace: testNamespace,
	}, csiDaemonSet)
	require.NoError(t, err)
}

func Test_ConfigureCSIDriver_Disable(t *testing.T) {
	mapperEntries := map[string]string{
		testDynakube: "",
	}
	fakeClient := prepareFakeClientWithEnabledCSI(mapperEntries)
	rec := prepareReconciliation(false)

	err := ConfigureCSIDriver(fakeClient, scheme.Scheme, testOperatorPodName, testNamespace, rec, 10)
	require.NoError(t, err)

	updatedConfigMap := &corev1.ConfigMap{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      CsiMapperConfigMapName,
		Namespace: testNamespace,
	}, updatedConfigMap)
	require.NoError(t, err)
	assert.Nil(t, updatedConfigMap.Data)

	updatedDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      DaemonSetName,
		Namespace: testNamespace,
	}, updatedDaemonSet)
	require.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
}

func Test_ConfigureCSIDriver_RemoveDynakubeFromDisabledCSI(t *testing.T) {
	mapperEntries := map[string]string{
		testDynakube:     "",
		"other-dynakube": "",
	}
	fakeClient := prepareFakeClientWithEnabledCSI(mapperEntries)
	rec := prepareReconciliation(false)

	err := ConfigureCSIDriver(fakeClient, scheme.Scheme, testOperatorPodName, testNamespace, rec, 10)
	require.NoError(t, err)

	updatedConfigMap := &corev1.ConfigMap{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      CsiMapperConfigMapName,
		Namespace: testNamespace,
	}, updatedConfigMap)
	require.NoError(t, err)

	assert.NotNil(t, updatedConfigMap.Data)
	assert.Len(t, updatedConfigMap.Data, 1)
	assert.NotContains(t, updatedConfigMap.Data, testDynakube)

	updatedDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      DaemonSetName,
		Namespace: testNamespace,
	}, updatedDaemonSet)
	require.NoError(t, err)
}

func Test_ConfigureCSIDriver_AddDynakubeToEnabledCSI(t *testing.T) {
	mapperEntries := map[string]string{
		"other-dynakube": "",
	}
	fakeClient := prepareFakeClientWithEnabledCSI(mapperEntries)
	rec := prepareReconciliation(true)

	err := ConfigureCSIDriver(fakeClient, scheme.Scheme, testOperatorPodName, testNamespace, rec, 10)
	require.NoError(t, err)

	updatedConfigMap := &corev1.ConfigMap{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      CsiMapperConfigMapName,
		Namespace: testNamespace,
	}, updatedConfigMap)
	require.NoError(t, err)

	assert.NotNil(t, updatedConfigMap.Data)
	assert.Len(t, updatedConfigMap.Data, 2)
	assert.Contains(t, updatedConfigMap.Data, testDynakube)
	assert.Equal(t, "", updatedConfigMap.Data[testDynakube])

	updatedDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Name:      DaemonSetName,
		Namespace: testNamespace,
	}, updatedDaemonSet)
	require.NoError(t, err)
}

func prepareFakeClient(objs ...client.Object) client.Client {
	objs = append(objs,
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testOperatorPodName,
				Namespace: testNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: testOperatorImage,
					},
				},
			},
		})
	return fake.NewClient(objs...)
}

func prepareFakeClientWithEnabledCSI(data map[string]string) client.Client {
	csiDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      DaemonSetName,
		},
	}
	mapperConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      CsiMapperConfigMapName,
		},
		Data: data,
	}
	fakeClient := prepareFakeClient(csiDaemonSet, mapperConfigMap)
	return fakeClient
}

func prepareReconciliation(enableCodeModules bool) *utils.Reconciliation {
	log := logger.NewDTLogger()
	rec := &utils.Reconciliation{
		Log: log,
		Instance: &v1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespace,
			},
			Spec: v1alpha1.DynaKubeSpec{
				CodeModules: v1alpha1.CodeModulesSpec{
					Enabled: enableCodeModules,
				},
			},
		},
	}
	return rec
}
