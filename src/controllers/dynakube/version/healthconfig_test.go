package version

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/src/oci/registry/mocks"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	fakecontainer "github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetOneAgentHealthConfig(t *testing.T) {
	dynakube := &dynatracev1beta1.DynaKube{}
	apiReader := fakek8s.NewClientBuilder().Build()
	pullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: dynakube.PullSecretName(),
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte(""),
		},
	}
	apiReader.Create(context.Background(), pullSecret)

	imageUri := "testImage"
	interval := time.Second * 10
	timeout := time.Second * 30
	startPeriod := time.Second * 1200
	retries := 3

	t.Run("get healthConfig with test as CMD", func(t *testing.T) {
		testCommands := []string{"CMD", "echo", "test"}
		dynakube := &dynatracev1beta1.DynaKube{}
		fakeImage := &fakecontainer.FakeImage{}
		fakeImage.ConfigFileStub = func() (*containerv1.ConfigFile, error) {
			return &containerv1.ConfigFile{
				Config: containerv1.Config{
					Healthcheck: &containerv1.HealthConfig{
						Test:        testCommands,
						Interval:    interval,
						Timeout:     timeout,
						StartPeriod: startPeriod,
						Retries:     retries,
					},
				},
			}, nil
		}

		fakeImage.ConfigFile()
		image := containerv1.Image(fakeImage)

		registryClient := &mocks.MockImageGetter{}
		registryClient.On("PullImageInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&image, nil)
		healthConfig, err := GetOneAgentHealthConfig(context.Background(), apiReader, registryClient, dynakube, imageUri)

		assert.Nil(t, err)
		assert.NotNil(t, healthConfig)
		assert.Equal(t, testCommands[1:], healthConfig.Test)
		assert.Equal(t, interval, healthConfig.Interval)
		assert.Equal(t, timeout, healthConfig.Timeout)
		assert.Equal(t, startPeriod, healthConfig.StartPeriod)
		assert.Equal(t, retries, healthConfig.Retries)
	})
	t.Run("get healthConfig with test as CMD-SHELL", func(t *testing.T) {
		testCommands := []string{"CMD-SHELL", "echo", "test"}
		dynakube := &dynatracev1beta1.DynaKube{}
		fakeImage := &fakecontainer.FakeImage{}
		fakeImage.ConfigFileStub = func() (*containerv1.ConfigFile, error) {
			return &containerv1.ConfigFile{
				Config: containerv1.Config{
					Healthcheck: &containerv1.HealthConfig{
						Test:        testCommands,
						Interval:    interval,
						Timeout:     timeout,
						StartPeriod: startPeriod,
						Retries:     retries,
					},
				},
			}, nil
		}

		fakeImage.ConfigFile()
		image := containerv1.Image(fakeImage)

		registryClient := &mocks.MockImageGetter{}
		registryClient.On("PullImageInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&image, nil)
		healthConfig, err := GetOneAgentHealthConfig(context.Background(), apiReader, registryClient, dynakube, imageUri)

		expectedTestCommands := append([]string{"/bin/sh", "-c"}, testCommands[1:]...)

		assert.Nil(t, err)
		assert.NotNil(t, healthConfig)
		assert.Equal(t, expectedTestCommands, healthConfig.Test)
		assert.Equal(t, interval, healthConfig.Interval)
		assert.Equal(t, timeout, healthConfig.Timeout)
		assert.Equal(t, startPeriod, healthConfig.StartPeriod)
		assert.Equal(t, retries, healthConfig.Retries)
	})
	t.Run("get healthConfig with default values", func(t *testing.T) {
		testCommands := []string{"CMD", "echo", "test"}
		dynakube := &dynatracev1beta1.DynaKube{}
		fakeImage := &fakecontainer.FakeImage{}
		fakeImage.ConfigFileStub = func() (*containerv1.ConfigFile, error) {
			return &containerv1.ConfigFile{
				Config: containerv1.Config{
					Healthcheck: &containerv1.HealthConfig{
						Test: testCommands,
					},
				},
			}, nil
		}

		fakeImage.ConfigFile()
		image := containerv1.Image(fakeImage)

		registryClient := &mocks.MockImageGetter{}
		registryClient.On("PullImageInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&image, nil)
		healthConfig, err := GetOneAgentHealthConfig(context.Background(), apiReader, registryClient, dynakube, imageUri)

		assert.Nil(t, err)
		assert.NotNil(t, healthConfig)
		assert.Equal(t, testCommands[1:], healthConfig.Test)
		assert.Equal(t, DefaultHealthConfigInterval, healthConfig.Interval)
		assert.Equal(t, DefaultHealthConfigTimeout, healthConfig.Timeout)
		assert.Equal(t, DefaultHealthConfigStartPeriod, healthConfig.StartPeriod)
		assert.Equal(t, DefaultHealthConfigRetries, healthConfig.Retries)
	})
	t.Run("healthConfig is not existent in OneAgent image", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{}
		fakeImage := &fakecontainer.FakeImage{}
		fakeImage.ConfigFileStub = func() (*containerv1.ConfigFile, error) {
			return &containerv1.ConfigFile{
				Config: containerv1.Config{},
			}, nil
		}

		fakeImage.ConfigFile()
		image := containerv1.Image(fakeImage)

		registryClient := &mocks.MockImageGetter{}
		registryClient.On("PullImageInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&image, nil)
		healthConfig, err := GetOneAgentHealthConfig(context.Background(), apiReader, registryClient, dynakube, imageUri)

		assert.Nil(t, err)
		assert.Nil(t, healthConfig)
	})
}
