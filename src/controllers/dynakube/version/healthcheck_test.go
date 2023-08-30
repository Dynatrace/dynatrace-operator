package version

import (
	"context"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/registry/mocks"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	fakecontainer "github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSetOneAgentHealthcheck(t *testing.T) {
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

	t.Run("docker image contains healthcheck property as CMD", func(t *testing.T) {
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
		err := SetOneAgentHealthcheck(context.Background(), apiReader, registryClient, dynakube, imageUri)

		assert.Nil(t, err)
		assert.NotNil(t, dynakube.Status.OneAgent.Healthcheck)
		assert.Equal(t, testCommands[1:], dynakube.Status.OneAgent.Healthcheck.Test)
		assert.Equal(t, interval, dynakube.Status.OneAgent.Healthcheck.Interval)
		assert.Equal(t, timeout, dynakube.Status.OneAgent.Healthcheck.Timeout)
		assert.Equal(t, startPeriod, dynakube.Status.OneAgent.Healthcheck.StartPeriod)
		assert.Equal(t, retries, dynakube.Status.OneAgent.Healthcheck.Retries)
	})
	t.Run("docker image contains healthcheck property as CMD-SHELL", func(t *testing.T) {
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
		err := SetOneAgentHealthcheck(context.Background(), apiReader, registryClient, dynakube, imageUri)

		expectedTestCommands := append([]string{"/bin/sh", "-c"}, testCommands[1:]...)

		assert.Nil(t, err)
		assert.NotNil(t, dynakube.Status.OneAgent.Healthcheck)
		assert.Equal(t, expectedTestCommands, dynakube.Status.OneAgent.Healthcheck.Test)
		assert.Equal(t, interval, dynakube.Status.OneAgent.Healthcheck.Interval)
		assert.Equal(t, timeout, dynakube.Status.OneAgent.Healthcheck.Timeout)
		assert.Equal(t, startPeriod, dynakube.Status.OneAgent.Healthcheck.StartPeriod)
		assert.Equal(t, retries, dynakube.Status.OneAgent.Healthcheck.Retries)
	})
	t.Run("docker image contains healthcheck test but no interval-timeout-etc", func(t *testing.T) {
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
		err := SetOneAgentHealthcheck(context.Background(), apiReader, registryClient, dynakube, imageUri)

		assert.Nil(t, err)
		assert.NotNil(t, dynakube.Status.OneAgent.Healthcheck)
		assert.Equal(t, testCommands[1:], dynakube.Status.OneAgent.Healthcheck.Test)
		assert.Equal(t, DefaultHealthConfigInterval, dynakube.Status.OneAgent.Healthcheck.Interval)
		assert.Equal(t, DefaultHealthConfigTimeout, dynakube.Status.OneAgent.Healthcheck.Timeout)
		assert.Equal(t, DefaultHealthConfigStartPeriod, dynakube.Status.OneAgent.Healthcheck.StartPeriod)
		assert.Equal(t, DefaultHealthConfigRetries, dynakube.Status.OneAgent.Healthcheck.Retries)
	})
	t.Run("docker image doesn't contain healthcheck property", func(t *testing.T) {
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
		err := SetOneAgentHealthcheck(context.Background(), apiReader, registryClient, dynakube, imageUri)

		assert.Nil(t, err)
		assert.Nil(t, dynakube.Status.OneAgent.Healthcheck)
	})
}
