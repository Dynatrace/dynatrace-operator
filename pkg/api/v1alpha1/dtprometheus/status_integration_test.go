// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus_test

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPrometheusMonitoringName = "dtprometheus"
	testNamespaceDtp             = "dynatrace"

	dummyConditionTypeDtp    = "dummyType"
	dummyConditionReasonDtp  = "dummyReason"
	dummyConditionMessageDtp = "dummyMessage"

	duplicatedConditionErrorMessageDtp = `PrometheusMonitoring.dynatrace.com "dtprometheus" is invalid: status.conditions[1]: Duplicate value: {"type":"dummyType"}`
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
		dtp := buildPrometheusMonitoring()
		createPrometheusMonitoring(t, clt, dtp)
		dummyCondition := buildPrometheusMonitoringCondition()

		// append first condition
		*dtp.Conditions() = append(*dtp.Conditions(), dummyCondition)
		require.NoError(t, clt.Status().Update(t.Context(), dtp))

		// check that condition was added
		clt.Get(t.Context(), client.ObjectKeyFromObject(dtp), dtp)
		require.Len(t, *dtp.Conditions(), 1)

		// append duplicated condition
		*dtp.Conditions() = append(*dtp.Conditions(), dummyCondition)
		require.ErrorContains(t, clt.Status().Update(t.Context(), dtp), duplicatedConditionErrorMessageDtp)

		// check that condition count is still 1
		clt.Get(t.Context(), client.ObjectKeyFromObject(dtp), dtp)
		require.Len(t, *dtp.Conditions(), 1)
	})
}

func buildPrometheusMonitoring() *dtprometheus.PrometheusMonitoring {
	return &dtprometheus.PrometheusMonitoring{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testPrometheusMonitoringName,
			Namespace:   testNamespaceDtp,
			Annotations: map[string]string{},
		},
		Spec:   dtprometheus.PrometheusMonitoringSpec{DynaKubeRef: "dynakube"},
		Status: dtprometheus.PrometheusMonitoringStatus{},
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

func createPrometheusMonitoring(t *testing.T, clt client.Client, dtp *dtprometheus.PrometheusMonitoring) {
	t.Helper()
	status := dtp.Status
	integrationtests.CreateKubernetesObject(t, clt, dtp)
	dtp.Status = status
	require.NoError(t, clt.Status().Update(t.Context(), dtp))
}
