package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
)

func TestBuildLabels(t *testing.T) {
	l := BuildLabels("my-name", DeploymentTypeFullStack)
	assert.Equal(t, l[kubeobjects.AppComponentLabel], string(kubeobjects.OneAgentComponentLabel))
	assert.Equal(t, l[kubeobjects.AppCreatedByLabel], "my-name")
	assert.Equal(t, l[kubeobjects.FeatureLabel], DeploymentTypeFullStack)
}
