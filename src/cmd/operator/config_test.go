package operator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/rest"
)

type mockConfigProvider struct {
	mock.Mock
}

func (provider *mockConfigProvider) GetConfig() (*rest.Config, error) {
	args := provider.Called()
	return args.Get(0).(*rest.Config), args.Error(1)
}

func TestConfigProvider(t *testing.T) {
	provider := newKubeConfigProvider()

	assert.NotNil(t, provider)
}
