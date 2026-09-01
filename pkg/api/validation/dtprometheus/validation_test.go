// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var testDTPrometheus = &dtprometheus.DTPrometheus{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-name",
		Namespace: "test-namespace",
	},
	Spec: dtprometheus.DTPrometheusSpec{
		DynaKubeName: "test-dynakube",
	},
}

var noPrometheusCRDsInterceptor = interceptor.Funcs{
	List: func(_ context.Context, _ client.WithWatch, _ client.ObjectList, _ ...client.ListOption) error {
		return &meta.NoKindMatchError{}
	},
}

func TestMissingPrometheusCRDs(t *testing.T) {
	t.Run("all CRDs missing produces a single warning listing all of them", func(t *testing.T) {
		clt := fake.NewClientWithInterceptors(noPrometheusCRDsInterceptor)

		msg := missingPrometheusCRDs(t.Context(), clt)

		assert.Contains(t, msg, "servicemonitors.monitoring.coreos.com")
		assert.Contains(t, msg, "podmonitors.monitoring.coreos.com")
		assert.Contains(t, msg, "probes.monitoring.coreos.com")
		assert.Contains(t, msg, "scrapeconfigs.monitoring.coreos.com")
	})

	t.Run("only some CRDs missing lists just those", func(t *testing.T) {
		partialInterceptor := interceptor.Funcs{
			List: func(_ context.Context, _ client.WithWatch, list client.ObjectList, _ ...client.ListOption) error {
				if list.GetObjectKind().GroupVersionKind().Kind == "ScrapeConfigList" {
					return &meta.NoKindMatchError{}
				}

				return nil
			},
		}
		clt := fake.NewClientWithInterceptors(partialInterceptor)

		msg := missingPrometheusCRDs(t.Context(), clt)

		assert.Contains(t, msg, "scrapeconfigs.monitoring.coreos.com")
		assert.NotContains(t, msg, "servicemonitors.monitoring.coreos.com")
		assert.NotContains(t, msg, "podmonitors.monitoring.coreos.com")
		assert.NotContains(t, msg, "probes.monitoring.coreos.com")
	})

	t.Run("all CRDs present produces no warning", func(t *testing.T) {
		clt := fake.NewClient()

		msg := missingPrometheusCRDs(t.Context(), clt)

		assert.Empty(t, msg)
	})

	t.Run("non-NoMatch error assumes the CRD is present and produces no warning", func(t *testing.T) {
		forbiddenInterceptor := interceptor.Funcs{
			List: func(_ context.Context, _ client.WithWatch, list client.ObjectList, _ ...client.ListOption) error {
				return apierrors.NewForbidden(schema.GroupResource{Group: prometheusOperatorGroup, Resource: list.GetObjectKind().GroupVersionKind().Kind}, "", nil)
			},
		}
		clt := fake.NewClientWithInterceptors(forbiddenInterceptor)

		msg := missingPrometheusCRDs(t.Context(), clt)

		assert.Empty(t, msg)
	})
}

func TestValidateCreateAndUpdateSurfaceWarning(t *testing.T) {
	clt := fake.NewClientWithInterceptors(noPrometheusCRDsInterceptor)
	validator := &Validator{apiReader: clt}

	t.Run("ValidateCreate", func(t *testing.T) {
		warnings, err := validator.ValidateCreate(t.Context(), testDTPrometheus)

		require.NoError(t, err)
		require.Len(t, warnings, 1)
	})

	t.Run("ValidateUpdate", func(t *testing.T) {
		warnings, err := validator.ValidateUpdate(t.Context(), testDTPrometheus, testDTPrometheus)

		require.NoError(t, err)
		require.Len(t, warnings, 1)
	})
}

func TestValidateCreateAndUpdateWithAllCRDsPresent(t *testing.T) {
	clt := fake.NewClient()
	validator := &Validator{apiReader: clt}

	warnings, err := validator.ValidateCreate(t.Context(), testDTPrometheus)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	warnings, err = validator.ValidateUpdate(t.Context(), testDTPrometheus, testDTPrometheus)
	require.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestNew(t *testing.T) {
	clt := fake.NewClient()
	v := New(clt)

	validator, ok := v.(*Validator)
	require.True(t, ok)
	assert.Equal(t, clt, validator.apiReader)
}

func TestValidateDelete(t *testing.T) {
	validator := &Validator{}
	warnings, err := validator.ValidateDelete(t.Context(), testDTPrometheus)
	assert.Nil(t, warnings)
	assert.NoError(t, err)
}

func TestValidateCreateAndUpdateWithWrongType(t *testing.T) {
	clt := fake.NewClientWithInterceptors(noPrometheusCRDsInterceptor)
	validator := &Validator{apiReader: clt}

	// no GVK set, hits "unknown object %T"
	noGVK := &corev1.Pod{}
	// GVK set but wrong kind, hits "unknown object %s"
	withGVK := &corev1.Pod{TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"}}

	_, err := validator.ValidateCreate(t.Context(), noGVK)
	require.Error(t, err)

	_, err = validator.ValidateCreate(t.Context(), withGVK)
	require.Error(t, err)

	_, err = validator.ValidateUpdate(t.Context(), testDTPrometheus, noGVK)
	require.Error(t, err)
}
