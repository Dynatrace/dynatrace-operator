package validation

import (
	"context"
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
	_, err := runValidators(ec, other...)
	require.Error(t, err)

	for _, errMsg := range errMessages {
		assert.Contains(t, err.Error(), errMsg)
	}
}

func assertAllowed(t *testing.T, ec *edgeconnect.EdgeConnect, other ...client.Object) {
	warn, err := runValidators(ec, other...)
	require.NoError(t, err)
	assert.Empty(t, warn)
}

func runValidators(ec *edgeconnect.EdgeConnect, other ...client.Object) (admission.Warnings, error) {
	clt := fake.NewClient()
	if other != nil {
		clt = fake.NewClient(other...)
	}

	validator := &Validator{
		apiReader: clt,
		cfg:       &rest.Config{},
		modules:   installconfig.GetModules(),
	}

	return validator.ValidateCreate(context.Background(), ec)
}
