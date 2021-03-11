// +build e2e

package e2e

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// CreateClient creates a client from the local kube-config for e2e testing
func CreateClient(t *testing.T) client.Client {
	cfg, err := config.GetConfig()
	assert.NoError(t, err)

	err = dynatracev1alpha1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)

	clt, err := client.New(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
	assert.NotNil(t, clt)
	assert.NoError(t, err)

	return clt
}
