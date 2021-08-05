package webhook

import (
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func setupTestEnvironment(_ *testing.T) *envtest.Environment {
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "default", "bases")},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths:                    []string{filepath.Join("..", "config", "common", "webhook", "validation")},
			IgnoreErrorIfPathMissing: true,
		},
	}
	return testEnv
}

func TestAddDynakubeValidationWebhookToManager(t *testing.T) {
	testEnv := setupTestEnvironment(t)
	cfg, err := testEnv.Start()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		Host:               webhookInstallOptions.LocalServingHost,
		Port:               webhookInstallOptions.LocalServingPort,
		CertDir:            webhookInstallOptions.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	require.NoError(t, err)
	require.NotNil(t, mgr)

	err = AddDynakubeValidationWebhookToManager(mgr)
	assert.NoError(t, err)
}

func TestHasConflictingNodeSelectors(t *testing.T) {
	t.Run(`no conflicts`, func(t *testing.T) {
		dynakube := buildTestInstance(t)

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			"label1": "value1",
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			"label2": "value2",
		}
		assert.False(t, hasConflictingNodeSelectors(dynakube))

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			"label1": "value1",
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			"label1": "value2",
		}
		assert.False(t, hasConflictingNodeSelectors(dynakube))

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			"label1": "value2",
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			"label1": "value1",
		}
		assert.False(t, hasConflictingNodeSelectors(dynakube))
	})

	t.Run(`conflicting node selectors`, func(t *testing.T) {
		dynakube := buildTestInstance(t)

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			"label1": "value1",
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			"label1": "value1",
		}
		assert.True(t, hasConflictingNodeSelectors(dynakube))

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			// Empty map matches everything
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			"label1": "value1",
		}
		assert.True(t, hasConflictingNodeSelectors(dynakube))

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			"label1": "value1",
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			// Empty map matches everything
		}
		assert.True(t, hasConflictingNodeSelectors(dynakube))

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			// Empty map matches everything
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			// Empty map matches everything
		}
		assert.True(t, hasConflictingNodeSelectors(dynakube))

	})
}

func buildTestInstance(_ *testing.T) v1alpha1.DynaKube {
	return v1alpha1.DynaKube{
		Spec: v1alpha1.DynaKubeSpec{
			ClassicFullStack: v1alpha1.FullStackSpec{
				Enabled: true,
			},
			InfraMonitoring: v1alpha1.FullStackSpec{
				Enabled: true,
			},
		},
	}
}
