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
	testDTPrometheusName = "dtprometheus"
	testNamespaceDtp     = "dynatrace"

	dummyConditionTypeDtp    = "dummyType"
	dummyConditionReasonDtp  = "dummyReason"
	dummyConditionMessageDtp = "dummyMessage"

	duplicatedConditionErrorMessageDtp = `DTPrometheus.dynatrace.com "dtprometheus" is invalid: status.conditions[1]: Duplicate value: {"type":"dummyType"}`
)

func TestDTPrometheusUpdateStatus(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	clt.Create(t.Context(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNamespaceDtp,
			Labels: map[string]string{},
		},
	})

	t.Run("can't add duplicated conditions", func(t *testing.T) {
		dtp := buildDTPrometheus()
		createDTPrometheus(t, clt, dtp)
		dummyCondition := buildDTPrometheusCondition()

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

func buildDTPrometheus() *dtprometheus.DTPrometheus {
	return &dtprometheus.DTPrometheus{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDTPrometheusName,
			Namespace:   testNamespaceDtp,
			Annotations: map[string]string{},
		},
		Spec:   dtprometheus.DTPrometheusSpec{DynaKubeName: "dynakube"},
		Status: dtprometheus.DTPrometheusStatus{},
	}
}

func buildDTPrometheusCondition() metav1.Condition {
	return metav1.Condition{
		Type:               dummyConditionTypeDtp,
		Status:             metav1.ConditionTrue,
		Reason:             dummyConditionReasonDtp,
		Message:            dummyConditionMessageDtp,
		LastTransitionTime: metav1.Now(),
	}
}

func createDTPrometheus(t *testing.T, clt client.Client, dtp *dtprometheus.DTPrometheus) {
	t.Helper()
	status := dtp.Status
	integrationtests.CreateKubernetesObject(t, clt, dtp)
	dtp.Status = status
	require.NoError(t, clt.Status().Update(t.Context(), dtp))
}
