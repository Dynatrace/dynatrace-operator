// +build e2e

package dynakubetests

import (
	"context"
	"fmt"
	"os"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	keyApiURL = "DYNATRACE_API_URL"

	namespace           = "dynatrace"
	maxWaitCycles       = 5
	trustedCertPath     = "/mnt/dynatrace/certs"
	trustedCertFilename = "certs.pem"

	testImage    = "test-image:latest"
	testName     = "test-name"
	testData     = "test-data"
	testCertName = "certs"
)

func prepareDefaultEnvironment(t *testing.T) (string, client.Client) {
	apiURL := os.Getenv(keyApiURL)
	assert.NotEmpty(t, apiURL, fmt.Sprintf("variable %s must be set", keyApiURL))

	clt := e2e.CreateClient(t)
	assert.NotNil(t, clt)

	err := e2e.PrepareEnvironment(clt, namespace)
	require.NoError(t, err)

	return apiURL, clt
}

func createMinimumViableOneAgent(apiURL string) dynatracev1alpha1.DynaKube {
	return dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      testName,
		},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: apiURL,
			Tokens: e2e.TokenSecretName,
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				Enabled: true,
			},
			OneAgent: dynatracev1alpha1.OneAgentSpec{
				Image: testImage,
			},
		},
	}
}

func deployOneAgent(t *testing.T, clt client.Client, oneAgent *dynatracev1alpha1.DynaKube) e2e.PhaseWait {
	err := clt.Create(context.TODO(), oneAgent)
	assert.NoError(t, err)

	phaseWait := e2e.NewOneAgentWaitConfiguration(t, clt, maxWaitCycles, namespace, testName)
	err = phaseWait.WaitForPhase(dynatracev1alpha1.Deploying)
	assert.NoError(t, err)

	return phaseWait
}

func findOneAgentPods(t *testing.T, clt client.Client) (*dynatracev1alpha1.DynaKube, *corev1.PodList) {
	instance := &dynatracev1alpha1.DynaKube{}
	err := clt.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: testName}, instance)
	assert.NoError(t, err)

	podList := &corev1.PodList{}
	listOps := []client.ListOption{
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(buildLabels(instance.Name)),
	}
	err = clt.List(context.TODO(), podList, listOps...)
	assert.NoError(t, err)

	return instance, podList
}

func buildLabels(name string) map[string]string {
	return map[string]string{
		"dynatrace.com/component":         "operator",
		"operator.dynatrace.com/feature":  "classic",
		"operator.dynatrace.com/instance": name,
	}
}
