package csi

import (
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
	"testing"
)

func TestCsiCommand(t *testing.T) {
	expectedError := errors.New("config provider error")
	mockConfigProvider := &config.MockProvider{}
	mockConfigProvider.On("GetConfig").Return(&rest.Config{}, expectedError)
	command := newCsiCommandBuilder().
		setConfigProvider(mockConfigProvider).
		build()

	err := command.RunE(command, make([]string, 0))

	assert.EqualError(t, err, expectedError.Error())
	mockConfigProvider.AssertCalled(t, "GetConfig")
}
