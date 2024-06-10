package conditions

import (
	"errors"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestIsKubeApiError(t *testing.T) {
	assert.False(t, IsKubeApiError(nil), "Nil error should not be a Kube API error")
	assert.False(t, IsKubeApiError(errors.New("Some error")), "Non-Kube API error should not be a Kube API error")
	assert.True(t, IsKubeApiError(k8serrors.NewBadRequest("Bad request")), "Kube API error should be a Kube API error")

	t.Run("wrapped error", func(t *testing.T) {
		var err error
		err = k8serrors.NewBadRequest("Bad request")
		err = pkgerrors.WithStack(err)
		assert.True(t, IsKubeApiError(err), "Kube API error should be a Kube API error even if wrapped")
	})
}
