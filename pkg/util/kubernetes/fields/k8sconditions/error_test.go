package k8sconditions

import (
	"errors"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestIsKubeApiError(t *testing.T) {
	assert.False(t, IsKubeAPIError(nil), "Nil error should not be a Kube API error")
	assert.False(t, IsKubeAPIError(errors.New("Some error")), "Non-Kube API error should not be a Kube API error")
	assert.True(t, IsKubeAPIError(k8serrors.NewBadRequest("Bad request")), "Kube API error should be a Kube API error")

	t.Run("wrapped error", func(t *testing.T) {
		var err error
		err = k8serrors.NewBadRequest("Bad request")
		err = pkgerrors.WithStack(err)
		assert.True(t, IsKubeAPIError(err), "Kube API error should be a Kube API error even if wrapped")
	})
}
