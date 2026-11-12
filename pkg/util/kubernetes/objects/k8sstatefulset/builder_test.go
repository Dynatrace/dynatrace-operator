// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sstatefulset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	testDeploymentName  = "deployment-as-owner-of-service"
	testStatefulSetName = "test-statefulset-name"
	testDynakubeName    = "dynakube"
	testVersion         = "1.0.0"
	testAppName         = "dynatrace-operator"
)

func createDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		Name: testDeploymentName,
	}
}

func TestBuild(t *testing.T) {
	t.Run("create StatefulSet", func(t *testing.T) {
		container := corev1.Container{}
		service, err := Build(
			createDeployment(), testStatefulSetName, container)
		require.NoError(t, err)
		require.Len(t, service.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, service.OwnerReferences[0].Name)
		assert.Equal(t, testStatefulSetName, service.Name)
		assert.Empty(t, service.Labels)
	})
	t.Run("create StatefulSet with basic container configuration", func(t *testing.T) {
		container := corev1.Container{Name: "nginx", Image: "nginx", Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 80}}}
		service, err := Build(
			createDeployment(), testStatefulSetName, container)
		require.NoError(t, err)
		require.Len(t, service.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, service.OwnerReferences[0].Name)
		assert.Equal(t, testStatefulSetName, service.Name)
		assert.Len(t, service.Spec.Template.Spec.Containers, 1)
		assert.Equal(t, "nginx", service.Spec.Template.Spec.Containers[0].Name)
	})
	t.Run("create StatefulSet with labels", func(t *testing.T) {
		container := corev1.Container{}
		labelName := "name"
		labelValue := "value"
		customLabels := map[string]string{
			labelName: labelValue,
		}
		labels := k8slabel.New(testAppName, testDynakubeName, testVersion)
		statefulSet, err := Build(
			createDeployment(), testStatefulSetName, container, SetAllLabels(labels.AsMap(), labels.AsSelector(), labels.AsSelector(), customLabels))
		require.NoError(t, err)
		require.Len(t, statefulSet.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, statefulSet.OwnerReferences[0].Name)
		assert.Equal(t, testStatefulSetName, statefulSet.Name)
		expectedSelector := map[string]string{
			k8slabel.AppName:      testAppName,
			k8slabel.AppInstance:  testDynakubeName,
			k8slabel.AppManagedBy: version.AppName,
		}
		expectedLabels := map[string]string{
			k8slabel.AppName:         testAppName,
			k8slabel.AppInstance:     testDynakubeName,
			k8slabel.AppManagedBy:    version.AppName,
			k8slabel.AppVersion:      testVersion,
			k8slabel.OperatorVersion: version.Version,
		}
		// only template labels get the custom labels merged in
		expectedTemplateLabels := map[string]string{
			k8slabel.AppName:      testAppName,
			k8slabel.AppInstance:  testDynakubeName,
			k8slabel.AppManagedBy: version.AppName,
			labelName:             labelValue,
		}

		assert.Equal(t, expectedLabels, statefulSet.Labels)
		assert.Equal(t, expectedSelector, statefulSet.Spec.Selector.MatchLabels)
		assert.Equal(t, expectedTemplateLabels, statefulSet.Spec.Template.Labels)
	})
}
