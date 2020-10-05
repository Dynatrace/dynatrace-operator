package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHandleSecretError(t *testing.T) {
	result, err := HandleSecretError(nil, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	err = &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
	result, err = HandleSecretError(nil, err, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = HandleSecretError(nil, fmt.Errorf("custom error"), nil)
	assert.Error(t, err)
	assert.NotNil(t, result)

	secret := corev1.Secret{}
	result, err = HandleSecretError(&secret, nil, nil)
	assert.Error(t, err)
	assert.NotNil(t, result)
}
