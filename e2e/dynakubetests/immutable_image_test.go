// +build e2e

package dynakubetests

import (
	"context"
	"fmt"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/e2e"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestImmutableImage(t *testing.T) {
	t.Run(`pull secret is created if image is unset`, func(t *testing.T) {
		apiURL, clt := prepareDefaultEnvironment(t)

		instance := dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      testName,
			},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL: apiURL,
				Tokens: e2e.TokenSecretName,
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					Enabled:           true,
					UseImmutableImage: true,
				},
			},
		}

		err := clt.Create(context.TODO(), &instance)
		assert.NoError(t, err)

		phaseWait := e2e.NewOneAgentWaitConfiguration(t, clt, maxWaitCycles, namespace, testName)
		err = phaseWait.WaitForPhase(dynatracev1alpha1.Deploying)
		assert.NoError(t, err)

		pullSecret := v1.Secret{}
		err = clt.Get(context.TODO(), client.ObjectKey{Name: buildPullSecretName(), Namespace: namespace}, &pullSecret)
		assert.NoError(t, err)
	})
	t.Run(`no pull secret exists if customPullSecret is set`, func(t *testing.T) {
		apiURL, clt := prepareDefaultEnvironment(t)

		instance := dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      testName,
			},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:           apiURL,
				Tokens:           e2e.TokenSecretName,
				CustomPullSecret: testName,
				OneAgent: dynatracev1alpha1.OneAgentSpec{
					Image: testImage,
				},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					Enabled:           true,
					UseImmutableImage: true,
				},
			},
		}

		err := clt.Create(context.TODO(), &instance)
		assert.NoError(t, err)

		phaseWait := e2e.NewOneAgentWaitConfiguration(t, clt, maxWaitCycles, namespace, testName)
		err = phaseWait.WaitForPhase(dynatracev1alpha1.Deploying)
		assert.NoError(t, err)

		pullSecret := v1.Secret{}
		err = clt.Get(context.TODO(), client.ObjectKey{Name: buildPullSecretName(), Namespace: namespace}, &pullSecret)
		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func buildPullSecretName() string {
	return fmt.Sprintf("%s-pull-secret", testName)
}
