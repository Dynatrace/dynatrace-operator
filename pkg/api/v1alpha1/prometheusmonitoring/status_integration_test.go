// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring_test

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPrometheusMonitoringName = "prometheusmonitoring"
	testNamespaceDtp             = "dynatrace"

	dummyConditionTypeDtp    = "dummyType"
	dummyConditionReasonDtp  = "dummyReason"
	dummyConditionMessageDtp = "dummyMessage"

	duplicatedConditionErrorMessageDtp = `PrometheusMonitoring.dynatrace.com "prometheusmonitoring" is invalid: status.conditions[1]: Duplicate value: {"type":"dummyType"}`
)

func TestPrometheusMonitoringUpdateStatus(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	clt.Create(t.Context(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespaceDtp,
			Labels: map[string]string{},
		},
	})

	t.Run("can't add duplicated conditions", func(t *testing.T) {
		pm := buildPrometheusMonitoring()
		createPrometheusMonitoring(t, clt, pm)
		dummyCondition := buildPrometheusMonitoringCondition()

		// append first condition
		*pm.Conditions() = append(*pm.Conditions(), dummyCondition)
		require.NoError(t, clt.Status().Update(t.Context(), pm))

		// check that condition was added
		clt.Get(t.Context(), client.ObjectKeyFromObject(pm), pm)
		require.Len(t, *pm.Conditions(), 1)

		// append duplicated condition
		*pm.Conditions() = append(*pm.Conditions(), dummyCondition)
		require.ErrorContains(t, clt.Status().Update(t.Context(), pm), duplicatedConditionErrorMessageDtp)

		// check that condition count is still 1
		clt.Get(t.Context(), client.ObjectKeyFromObject(pm), pm)
		require.Len(t, *pm.Conditions(), 1)
	})
}

func buildPrometheusMonitoring() *prometheusmonitoring.PrometheusMonitoring {
	return &prometheusmonitoring.PrometheusMonitoring{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testPrometheusMonitoringName,
			Namespace:   testNamespaceDtp,
			Annotations: map[string]string{},
		},
		Spec:   prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeRef: "dynakube"},
		Status: prometheusmonitoring.PrometheusMonitoringStatus{},
	}
}

func buildPrometheusMonitoringCondition() metav1.Condition {
	return metav1.Condition{
		Type:               dummyConditionTypeDtp,
		Status:             metav1.ConditionTrue,
		Reason:             dummyConditionReasonDtp,
		Message:            dummyConditionMessageDtp,
		LastTransitionTime: metav1.Now(),
	}
}

func createPrometheusMonitoring(t *testing.T, clt client.Client, pm *prometheusmonitoring.PrometheusMonitoring) {
	t.Helper()
	status := pm.Status
	integrationtests.CreateKubernetesObject(t, clt, pm)
	pm.Status = status
	require.NoError(t, clt.Status().Update(t.Context(), pm))
}
