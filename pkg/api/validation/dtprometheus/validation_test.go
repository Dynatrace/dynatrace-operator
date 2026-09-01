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
		warnings := collectWarnings(t, noPrometheusCRDsInterceptor, testDTPrometheus)

		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "servicemonitors.monitoring.coreos.com")
		assert.Contains(t, warnings[0], "podmonitors.monitoring.coreos.com")
		assert.Contains(t, warnings[0], "probes.monitoring.coreos.com")
		assert.Contains(t, warnings[0], "scrapeconfigs.monitoring.coreos.com")
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

		warnings := collectWarnings(t, partialInterceptor, testDTPrometheus)

		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "scrapeconfigs.monitoring.coreos.com")
		assert.NotContains(t, warnings[0], "servicemonitors.monitoring.coreos.com")
		assert.NotContains(t, warnings[0], "podmonitors.monitoring.coreos.com")
		assert.NotContains(t, warnings[0], "probes.monitoring.coreos.com")
	})

	t.Run("all CRDs present produces no warning", func(t *testing.T) {
		presentInterceptor := interceptor.Funcs{
			List: func(_ context.Context, _ client.WithWatch, _ client.ObjectList, _ ...client.ListOption) error {
				return nil
			},
		}

		warnings := collectWarnings(t, presentInterceptor, testDTPrometheus)

		assert.Empty(t, warnings)
	})

	t.Run("non-NoMatch error assumes the CRD is present and produces no warning", func(t *testing.T) {
		forbiddenInterceptor := interceptor.Funcs{
			List: func(_ context.Context, _ client.WithWatch, list client.ObjectList, _ ...client.ListOption) error {
				return apierrors.NewForbidden(schema.GroupResource{Group: prometheusOperatorGroup, Resource: list.GetObjectKind().GroupVersionKind().Kind}, "", nil)
			},
		}

		warnings := collectWarnings(t, forbiddenInterceptor, testDTPrometheus)

		assert.Empty(t, warnings)
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

func collectWarnings(t *testing.T, funcs interceptor.Funcs, dtp *dtprometheus.DTPrometheus) []string {
	t.Helper()

	clt := fake.NewClientWithInterceptors(funcs)
	validator := &Validator{apiReader: clt}

	return validator.runValidators(t.Context(), validatorWarningFuncs, dtp)
}
