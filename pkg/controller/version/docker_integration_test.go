//+build integration

package version

import (
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/parser"
	"github.com/stretchr/testify/assert"
)

func TestDockerVersionChecker_DockerHub(t *testing.T) {
	alpineTag := "alpine:3.12.0"
	alpineDigest := "alpine@sha256:185518070891758909c9f839cf4ca393ee977ac378609f700f60a771a2dfe321"
	alpineTagOther := "alpine:3.11.6"

	demoRepoTag := "michaelrynkiewicz/demo-repo:2.0.0"
	demoRepoDigest := "michaelrynkiewicz/demo-repo@sha256:10e11125048ef2990b21e836ca2483614dfa401d74a40bbb3445dbec8a803b83"
	demoRepoTagOther := "michaelrynkiewicz/demo-repo:1.0.0"

	dockerHubConfig := &parser.DockerConfig{
		Auths: map[string]struct {
			Username string
			Password string
		}{
			"https://docker.io": {
				Username: os.Getenv("DOCKER_USERNAME"),
				Password: os.Getenv("DOCKER_PASSWORD"),
			}},
	}

	dockerVersionChecker := NewDockerVersionChecker(
		alpineTag,
		alpineDigest,
		dockerHubConfig)
	isLatest, err := dockerVersionChecker.IsLatest()
	assert.True(t, isLatest)
	assert.NoError(t, err)

	dockerVersionChecker = NewDockerVersionChecker(
		alpineTagOther,
		alpineDigest,
		dockerHubConfig)
	isLatest, err = dockerVersionChecker.IsLatest()
	assert.False(t, isLatest)
	assert.NoError(t, err)

	dockerVersionChecker = NewDockerVersionChecker(
		demoRepoTag,
		demoRepoDigest,
		dockerHubConfig)
	isLatest, err = dockerVersionChecker.IsLatest()
	assert.True(t, isLatest)
	assert.NoError(t, err)

	dockerVersionChecker = NewDockerVersionChecker(
		demoRepoTagOther,
		demoRepoDigest,
		dockerHubConfig)
	isLatest, err = dockerVersionChecker.IsLatest()
	assert.False(t, isLatest)
	assert.NoError(t, err)
}

func TestDockerVersionChecker_DockerHub_ConfigNoProtocol(t *testing.T) {
	alpineTag := "alpine:3.12.0"
	alpineDigest := "alpine@sha256:185518070891758909c9f839cf4ca393ee977ac378609f700f60a771a2dfe321"
	alpineTagOther := "alpine:3.11.6"

	demoRepoTag := "michaelrynkiewicz/demo-repo:2.0.0"
	demoRepoDigest := "michaelrynkiewicz/demo-repo@sha256:10e11125048ef2990b21e836ca2483614dfa401d74a40bbb3445dbec8a803b83"
	demoRepoTagOther := "michaelrynkiewicz/demo-repo:1.0.0"

	dockerHubConfig := &parser.DockerConfig{
		Auths: map[string]struct {
			Username string
			Password string
		}{
			"docker.io": {
				Username: os.Getenv("DOCKER_USERNAME"),
				Password: os.Getenv("DOCKER_PASSWORD"),
			}},
	}

	dockerVersionChecker := NewDockerVersionChecker(
		alpineTag,
		alpineDigest,
		dockerHubConfig)
	isLatest, err := dockerVersionChecker.IsLatest()
	assert.True(t, isLatest)
	assert.NoError(t, err)

	dockerVersionChecker = NewDockerVersionChecker(
		alpineTagOther,
		alpineDigest,
		dockerHubConfig)
	isLatest, err = dockerVersionChecker.IsLatest()
	assert.False(t, isLatest)
	assert.NoError(t, err)

	dockerVersionChecker = NewDockerVersionChecker(
		demoRepoTag,
		demoRepoDigest,
		dockerHubConfig)
	isLatest, err = dockerVersionChecker.IsLatest()
	assert.True(t, isLatest)
	assert.NoError(t, err)

	dockerVersionChecker = NewDockerVersionChecker(
		demoRepoTagOther,
		demoRepoDigest,
		dockerHubConfig)
	isLatest, err = dockerVersionChecker.IsLatest()
	assert.False(t, isLatest)
	assert.NoError(t, err)
}

func TestDockerVersionChecker_Quay(t *testing.T) {
	oneagentOperatorTag := "quay.io/dynatrace/dynatrace-oneagent-operator:v0.8.1"
	oneagentOperatorDigest := "quay.io/dynatrace/dynatrace-oneagent-operator@sha256:2713af0a484016e22a1cf0c925534e2c3c86670a829669d73295acec3d7688e3"
	oneagentOperatorTagOther := "quay.io/dynatrace/dynatrace-oneagent-operator:v0.6.0"

	quayConfig := &parser.DockerConfig{
		Auths: map[string]struct {
			Username string
			Password string
		}{
			"https://quay.io": {
				Username: os.Getenv("QUAY_USERNAME"),
				Password: os.Getenv("QUAY_PASSWORD"),
			}},
	}

	dockerVersionChecker := NewDockerVersionChecker(
		oneagentOperatorTag,
		oneagentOperatorDigest,
		quayConfig)
	isLatest, err := dockerVersionChecker.IsLatest()
	assert.True(t, isLatest)
	assert.NoError(t, err)

	dockerVersionChecker = NewDockerVersionChecker(
		oneagentOperatorTagOther,
		oneagentOperatorDigest,
		quayConfig)
	isLatest, err = dockerVersionChecker.IsLatest()
	assert.False(t, isLatest)
	assert.NoError(t, err)
}

// Deactivated for now since registry is misbehaving
// Uncomment code below until 10.09.2020 and try again
//func TestDockerVersionChecker_Dynatrace(t *testing.T) {
//	oneagentInstallerTagMew := "asj34817.dev.dynatracelabs.com/linux/oneagent:1.203.0"
//	oneagentInstallerTagOld := "asj34817.dev.dynatracelabs.com/linux/oneagent:1.201.2"
//	oneagentInstallerDigestMew := "asj34817.dev.dynatracelabs.com/linux/oneagent@sha256:ec40092339736bd22ff9fdc556f4683b489d27093342b22686637be638f95343"
//	dynatraceConfig := &parser.DockerConfig{
//		Auths: map[string]struct {
//			Username string
//			Password string
//		}{
//			"https://asj34817.dev.dynatracelabs.com": {
//				Username: os.Getenv("DYNATRACE_USERNAME"),
//				Password: os.Getenv("DYNATRACE_PASSWORD"),
//			}},
//	}
//
//	dockerVersionChecker := NewDockerVersionChecker(
//		oneagentInstallerTagMew,
//		oneagentInstallerDigestMew,
//		dynatraceConfig)
//	isLatest, err := dockerVersionChecker.IsLatest()
//	assert.True(t, isLatest)
//	assert.NoError(t, err)
//
//	dockerVersionChecker = NewDockerVersionChecker(
//		oneagentInstallerTagOld,
//		oneagentInstallerDigestMew,
//		dynatraceConfig)
//	isLatest, err = dockerVersionChecker.IsLatest()
//	assert.False(t, isLatest)
//	assert.NoError(t, err)
//}
