// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func assertDenied(t *testing.T, errMessages []string, ec *edgeconnect.EdgeConnect, other ...client.Object) {
	t.Helper()

	_, err := runValidators(t, ec, other...)
	require.Error(t, err)

	for _, errMsg := range errMessages {
		assert.Contains(t, err.Error(), errMsg)
	}
}

func assertAllowed(t *testing.T, ec *edgeconnect.EdgeConnect, other ...client.Object) {
	t.Helper()

	warn, err := runValidators(t, ec, other...)
	require.NoError(t, err)
	assert.Empty(t, warn)
}

func assertUpdateDenied(t *testing.T, errMessages []string, oldEC *edgeconnect.EdgeConnect, newEC *edgeconnect.EdgeConnect, other ...client.Object) {
	t.Helper()
	_, err := runUpdateValidators(t, oldEC, newEC, other...)
	require.Error(t, err)

	for _, errMsg := range errMessages {
		assert.Contains(t, err.Error(), errMsg)
	}
}

func runValidators(t *testing.T, ec *edgeconnect.EdgeConnect, other ...client.Object) (admission.Warnings, error) {
	t.Helper()

	clt := fake.NewClient()
	if other != nil {
		clt = fake.NewClient(other...)
	}

	validator := &Validator{
		apiReader: clt,
		cfg:       &rest.Config{},
		modules:   installconfig.GetModules(),
	}

	return validator.ValidateCreate(t.Context(), ec)
}

func runUpdateValidators(t *testing.T, oldEC, newEC *edgeconnect.EdgeConnect, other ...client.Object) (admission.Warnings, error) {
	t.Helper()
	clt := fake.NewClient()

	if other != nil {
		clt = fake.NewClient(other...)
	}

	validator := &Validator{
		apiReader: clt,
		modules:   installconfig.GetModules(),
	}

	return validator.ValidateUpdate(t.Context(), oldEC, newEC)
}
