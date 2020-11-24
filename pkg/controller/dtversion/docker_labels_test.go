package dtversion

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDockerLabelsChecker_IsLatest(t *testing.T) {
	dockerLabels := NewDockerLabelsChecker(testImage, map[string]string{VersionKey: testVersion}, nil)
	assert.NotNil(t, dockerLabels)

	t.Run(`IsLatest returns true if versions are equal`, func(t *testing.T) {
		mockImageInformation := &MockImageInformation{}
		dockerLabels.imageInformationConstructor = func(s string, config *DockerConfig) ImageInformation {
			return mockImageInformation
		}

		mockImageInformation.
			On("GetVersionLabel").
			Return(testVersion, nil)

		isLatest, err := dockerLabels.IsLatest()
		assert.NoError(t, err)
		assert.True(t, isLatest)
	})
	t.Run(`IsLatest returns false if versions are unequal`, func(t *testing.T) {
		mockImageInformation := &MockImageInformation{}
		dockerLabels.imageInformationConstructor = func(s string, config *DockerConfig) ImageInformation {
			return mockImageInformation
		}

		mockImageInformation.
			On("GetVersionLabel").
			Return("2.0.0", nil)

		isLatest, err := dockerLabels.IsLatest()
		assert.NoError(t, err)
		assert.False(t, isLatest)

		mockImageInformation = &MockImageInformation{}
		dockerLabels.imageInformationConstructor = func(s string, config *DockerConfig) ImageInformation {
			return mockImageInformation
		}

		mockImageInformation.
			On("GetVersionLabel").
			Return("0.0.0", nil)

		isLatest, err = dockerLabels.IsLatest()
		assert.NoError(t, err)
		assert.False(t, isLatest)
	})
	t.Run(`IsLatest returns false and error on invalid label`, func(t *testing.T) {
		dockerLabels.labels = nil
		isLatest, err := dockerLabels.IsLatest()
		assert.False(t, isLatest)
		assert.Error(t, err)
	})
}
