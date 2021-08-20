package validation

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
	testApiUrl    = "https://f.q.d.n/api"
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

func TestDynakubeValidator_Handle(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {
		assertAllowedResponse(t, v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: testApiUrl,
				ClassicFullStack: v1alpha1.FullStackSpec{
					Enabled: false,
				},
				InfraMonitoring: v1alpha1.FullStackSpec{
					Enabled: false,
				},
			},
		})

		assertAllowedResponse(t, v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: testApiUrl,
				ClassicFullStack: v1alpha1.FullStackSpec{
					Enabled: true,
				},
				InfraMonitoring: v1alpha1.FullStackSpec{
					Enabled: false,
				},
			},
		})

		assertAllowedResponse(t, v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: testApiUrl,
				ClassicFullStack: v1alpha1.FullStackSpec{
					Enabled: false,
				},
				InfraMonitoring: v1alpha1.FullStackSpec{
					Enabled: true,
				},
			},
		})

		assertAllowedResponse(t, v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: testApiUrl,
				ClassicFullStack: v1alpha1.FullStackSpec{
					Enabled: true,
					NodeSelector: map[string]string{
						"label1": "value1",
					},
				},
				InfraMonitoring: v1alpha1.FullStackSpec{
					Enabled: true,
					NodeSelector: map[string]string{
						"label2": "value1",
					},
				},
			},
		})

		assertAllowedResponse(t, v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: testApiUrl,
				ClassicFullStack: v1alpha1.FullStackSpec{
					Enabled: true,
					NodeSelector: map[string]string{
						"label1": "value1",
					},
				},
				InfraMonitoring: v1alpha1.FullStackSpec{
					Enabled: true,
					NodeSelector: map[string]string{
						"label1": "value2",
					},
				},
			},
		})
	})
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t, v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: testApiUrl,
				ClassicFullStack: v1alpha1.FullStackSpec{
					Enabled: true,
				},
				InfraMonitoring: v1alpha1.FullStackSpec{
					Enabled: true,
				},
			},
		}, errorConflictingInfraMonitoringAndClassicNodeSelectors)

		assertDeniedResponse(t, v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: testApiUrl,
				ClassicFullStack: v1alpha1.FullStackSpec{
					Enabled: true,
					NodeSelector: map[string]string{
						"label1": "value1",
					},
				},
				InfraMonitoring: v1alpha1.FullStackSpec{
					Enabled: true,
					NodeSelector: map[string]string{
						"label1": "value1",
					},
				},
			},
		}, errorConflictingInfraMonitoringAndClassicNodeSelectors)

		assertDeniedResponse(t, v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: testApiUrl,
				ClassicFullStack: v1alpha1.FullStackSpec{
					Enabled: true,
					NodeSelector: map[string]string{
						"label1": "value1",
						"label2": "value2",
					},
				},
				InfraMonitoring: v1alpha1.FullStackSpec{
					Enabled: true,
					NodeSelector: map[string]string{
						"label1": "value1",
					},
				},
			},
		}, errorConflictingInfraMonitoringAndClassicNodeSelectors)
	})
	t.Run(`missing API URL`, func(t *testing.T) {
		assertDeniedResponse(t, v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: "",
				InfraMonitoring: v1alpha1.FullStackSpec{
					Enabled: true,
				},
			},
		}, errorNoApiUrl)
	})
	t.Run(`invalid API URL`, func(t *testing.T) {
		assertDeniedResponse(t, v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: exampleApiUrl,
				InfraMonitoring: v1alpha1.FullStackSpec{
					Enabled: true,
				},
			},
		}, errorNoApiUrl)
	})
}

func assertDeniedResponse(t *testing.T, dynakube v1alpha1.DynaKube, reason string) {
	response := handleRequest(t, dynakube)
	assert.False(t, response.Allowed)
	assert.Equal(t, metav1.StatusReason(reason), response.Result.Reason)
}

func assertAllowedResponse(t *testing.T, dynakube v1alpha1.DynaKube) {
	response := handleRequest(t, dynakube)
	assert.True(t, response.Allowed)
}

func handleRequest(t *testing.T, dynakube v1alpha1.DynaKube) admission.Response {
	clt := fake.NewClient()
	validator := &dynakubeValidator{
		logger: logger.NewDTLogger(),
		clt:    clt,
	}

	data, err := json.Marshal(dynakube)
	require.NoError(t, err)

	return validator.Handle(context.TODO(), admission.Request{
		AdmissionRequest: v1.AdmissionRequest{
			Name:      testName,
			Namespace: testNamespace,
			Object:    runtime.RawExtension{Raw: data},
		},
	})
}

func TestDynakubeValidator_InjectClient(t *testing.T) {
	validator := &dynakubeValidator{}
	clt := fake.NewClient()
	err := validator.InjectClient(clt)

	assert.NoError(t, err)
	assert.NotNil(t, validator.clt)
	assert.Equal(t, clt, validator.clt)
}

func TestHasConflictingConfiguration(t *testing.T) {
	t.Run(`no conflicts`, func(t *testing.T) {
		dynakube := buildTestInstance(t)

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			"label1": "value1",
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			"label2": "value2",
		}
		assert.False(t, hasConflictingConfiguration(dynakube))

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			"label1": "value1",
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			"label1": "value2",
		}
		assert.False(t, hasConflictingConfiguration(dynakube))

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			"label1": "value2",
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			"label1": "value1",
		}
		assert.False(t, hasConflictingConfiguration(dynakube))
	})

	t.Run(`conflicting node selectors`, func(t *testing.T) {
		dynakube := buildTestInstance(t)

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			"label1": "value1",
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			"label1": "value1",
		}
		assert.True(t, hasConflictingConfiguration(dynakube))

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			// Empty map matches everything
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			"label1": "value1",
		}
		assert.True(t, hasConflictingConfiguration(dynakube))

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			"label1": "value1",
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			// Empty map matches everything
		}
		assert.True(t, hasConflictingConfiguration(dynakube))

		dynakube.Spec.InfraMonitoring.NodeSelector = map[string]string{
			// Empty map matches everything
		}
		dynakube.Spec.ClassicFullStack.NodeSelector = map[string]string{
			// Empty map matches everything
		}
		assert.True(t, hasConflictingConfiguration(dynakube))
	})
}

func buildTestInstance(_ *testing.T) v1alpha1.DynaKube {
	return v1alpha1.DynaKube{
		Spec: v1alpha1.DynaKubeSpec{
			APIURL: testApiUrl,
			ClassicFullStack: v1alpha1.FullStackSpec{
				Enabled: true,
			},
			InfraMonitoring: v1alpha1.FullStackSpec{
				Enabled: true,
			},
		},
	}
}

func TestHasApiUrl(t *testing.T) {
	instance := v1alpha1.DynaKube{}
	assert.False(t, hasApiUrl(instance))

	instance.Spec.APIURL = testApiUrl
	assert.True(t, hasApiUrl(instance))
}
