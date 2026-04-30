package injection

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	oneagentclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	tokenclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	versions "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/otlp/exporterconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	oneagentclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/oneagent"
	settingsmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/settings"
	versionclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/version"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	versionmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testPaasToken       = "test-paas-token"
	testAPIToken        = "test-api-token"
	testDataIngestToken = "test-ingest-token"

	testUUID                  = "test-uuid"
	testTenantToken           = "abcd"
	testCommunicationEndpoint = "https://tenant.dev.dynatracelabs.com:443"

	testHost = "test-host"

	testDynakube   = "test-dynakube"
	testDynakube2  = "test-dynakube2"
	testNamespace  = "test-namespace"
	testNamespace2 = "test-namespace2"

	testNamespaceSelectorLabel = "namespaceSelector"

	testNamespaceDynatrace = "dynatrace"

	testAPIURL = "https://" + testHost + "/e/" + testUUID + "/api"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func TestReconciler(t *testing.T) {
	t.Run("add injection", func(t *testing.T) {
		expectedOneAgentConnectionInfo := oneagentclient.ConnectionInfo{
			TenantUUID:  testUUID,
			TenantToken: testTenantToken,
			Endpoints:   testCommunicationEndpoint,
		}
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							NamespaceSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									testNamespaceSelectorLabel: testDynakube,
								},
							},
						},
					},
				},
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							testNamespaceSelectorLabel: testDynakube,
						},
					},
				},
				OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							testNamespaceSelectorLabel: testDynakube,
						},
					},
					Signals: otlp.SignalConfiguration{
						Metrics: &otlp.MetricsSignal{},
					},
				},
			},
		}
		k8sconditions.SetOptionalScopeAvailable(dk.Conditions(), tokenclient.ConditionTypeAPITokenSettingsRead, "available")
		clt := fake.NewClientWithIndex(
			clientNotInjectedNamespace(testNamespace, testDynakube),
			clientNotInjectedNamespace(testNamespace2, testDynakube2),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				token.APIKey:        []byte(testAPIToken),
				token.PaaSKey:       []byte(testPaasToken),
				token.DataIngestKey: []byte(testDataIngestToken),
			}),
			dk,
		)
		oneAgentClient := oneagentclientmock.NewClient(t)
		oneAgentClient.EXPECT().GetConnectionInfo(t.Context()).Return(expectedOneAgentConnectionInfo, nil).Once()
		versionClient := versionclientmock.NewClient(t)
		versionClient.EXPECT().GetLatestAgentVersion(t.Context(), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("", nil)
		oneAgentClient.EXPECT().GetProcessModuleConfig(t.Context()).Return(&oneagentclient.ProcessModuleConfig{}, nil).Once()
		settingsClient := settingsmock.NewClient(t)
		settingsClient.EXPECT().GetRules(t.Context(), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, nil)
		dtClient := &dynatrace.Client{
			OneAgent: oneAgentClient,
			Settings: settingsClient,
			Version:  versionClient,
		}

		rec := NewReconciler(clt, clt)
		rec.istioReconciler = createIstioReconcilerMock(t, dk)

		err := rec.Reconcile(t.Context(), dtClient, dk)
		require.NoError(t, err)

		assertSecretFound(t, clt, dk.OneAgent().GetTenantSecret(), dk.Namespace)
		assertSecretFound(t, clt, consts.BootstrapperInitSecretName, testNamespace)
		assertSecretNotFound(t, clt, consts.BootstrapperInitSecretName, testNamespace2)

		assertSecretFound(t, clt, consts.OTLPExporterSecretName, testNamespace)
		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, testNamespace2)
	})
	t.Run("remove injection", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL:      testAPIURL,
				EnableIstio: true,
			},
		}
		setMetadataEnrichmentCreatedCondition(dk.Conditions())
		setCodeModulesInjectionCreatedCondition(dk.Conditions())

		clt := fake.NewClientWithIndex(
			clientInjectedNamespace(testNamespace, testDynakube),
			clientInjectedNamespace(testNamespace2, testDynakube2),
			clientSecret(consts.BootstrapperInitSecretName, testNamespace, nil),
			clientSecret(consts.BootstrapperInitSecretName, testNamespace2, nil),
			clientSecret(consts.BootstrapperInitCertsSecretName, testNamespace, nil),
			clientSecret(consts.BootstrapperInitCertsSecretName, testNamespace2, nil),
			clientSecret(consts.OTLPExporterSecretName, testNamespace, nil),
			clientSecret(consts.OTLPExporterSecretName, testNamespace2, nil),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				token.APIKey:  []byte(testAPIToken),
				token.PaaSKey: []byte(testPaasToken),
			}),
			dk,
		)
		settingsClient := settingsmock.NewClient(t)
		dtClient := &dynatrace.Client{Settings: settingsClient}

		rec := NewReconciler(clt, clt)
		rec.istioReconciler = createIstioReconcilerMock(t, dk)

		err := rec.Reconcile(t.Context(), dtClient, dk)
		require.NoError(t, err)

		assertSecretNotFound(t, clt, consts.BootstrapperInitSecretName, testNamespace)
		assertSecretFound(t, clt, consts.BootstrapperInitSecretName, testNamespace2)
		assertSecretNotFound(t, clt, consts.BootstrapperInitCertsSecretName, testNamespace)
		assertSecretFound(t, clt, consts.BootstrapperInitCertsSecretName, testNamespace2)

		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, testNamespace)
		assertSecretFound(t, clt, consts.OTLPExporterSecretName, testNamespace2)

		assert.Nil(t, meta.FindStatusCondition(*dk.Conditions(), metaDataEnrichmentConditionType))
		assert.Nil(t, meta.FindStatusCondition(*dk.Conditions(), codeModulesInjectionConditionType))
		assert.Nil(t, meta.FindStatusCondition(*dk.Conditions(), otlpExporterConfigurationConditionType))
	})
	t.Run("failure is logged in condition", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							NamespaceSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									testNamespaceSelectorLabel: testDynakube,
								},
							},
						},
					},
				},
			},
		}
		boomClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return k8serrors.NewInternalError(errors.New("test-error"))
			},
		})

		fakeReconciler := createReconcilerMock(t)
		fakeVersionReconciler := createVersionReconcilerMock(t)

		oneAgentClient := oneagentclientmock.NewClient(t)
		settingsClient := settingsmock.NewClient(t)
		dtClient := &dynatrace.Client{
			OneAgent: oneAgentClient,
			Settings: settingsClient}

		rec := NewReconciler(boomClient, boomClient)
		rec.istioReconciler = createIstioReconcilerMock(t, dk)
		rec.connectionInfoReconciler = fakeReconciler
		rec.versionReconciler = fakeVersionReconciler

		err := rec.Reconcile(t.Context(), dtClient, dk)
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), codeModulesInjectionConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func TestRemoveAppInjection(t *testing.T) {
	clt := clientRemoveAppInjection()
	dk := createDynaKube(testDynakube, testNamespaceDynatrace, oneagent.Spec{
		CloudNativeFullStack: nil,
	})
	rec := createReconciler(clt)

	rec.versionReconciler = createVersionReconcilerMock(t)
	rec.connectionInfoReconciler = createReconcilerMock(t)
	rec.enrichmentRulesReconciler = createReconcilerMock(t)
	rec.istioReconciler = createIstioReconcilerMock(t, dk)

	setCodeModulesInjectionCreatedCondition(dk.Conditions())
	setMetadataEnrichmentCreatedCondition(dk.Conditions())

	err := rec.Reconcile(t.Context(), &dynatrace.Client{}, dk)
	require.NoError(t, err)

	var namespace corev1.Namespace
	err = clt.Get(t.Context(), client.ObjectKey{Name: testNamespace, Namespace: ""}, &namespace)
	require.NoError(t, err)
	assert.Nil(t, namespace.Labels)

	err = clt.Get(t.Context(), client.ObjectKey{Name: testNamespace2, Namespace: ""}, &namespace)
	require.NoError(t, err)
	require.NotNil(t, namespace.Labels)
	assert.Equal(t, testDynakube2, namespace.Labels[dtwebhook.InjectionInstanceLabel])
	assert.Nil(t, namespace.Annotations)

	assertSecretNotFound(t, clt, consts.BootstrapperInitSecretName, testNamespace)
	assertSecretNotFound(t, clt, consts.BootstrapperInitSecretName, testNamespace2)
}

func TestSetupOneAgentInjection(t *testing.T) {
	t.Run("no injection - ClassicFullStack", func(t *testing.T) {
		clt := clientNoInjection()
		rec := createReconciler(clt)
		dk := createDynaKube(testDynakube, testNamespaceDynatrace, oneagent.Spec{
			ClassicFullStack: &oneagent.HostInjectSpec{},
		})
		rec.istioReconciler = createIstioReconcilerMock(t, dk)

		versionReconciler := createVersionReconcilerMock(t)
		connectionInfoReconciler := createReconcilerMock(t)

		err := rec.setupOneAgentInjection(t.Context(), dk, versionReconciler, connectionInfoReconciler)
		require.NoError(t, err)
	})

	t.Run("no injection - HostMonitoring", func(t *testing.T) {
		clt := clientNoInjection()
		rec := createReconciler(clt)
		dk := createDynaKube(testDynakube, testNamespaceDynatrace, oneagent.Spec{
			HostMonitoring: &oneagent.HostInjectSpec{},
		})
		rec.istioReconciler = createIstioReconcilerMock(t, dk)

		versionReconciler := createVersionReconcilerMock(t)
		connectionInfoReconciler := createReconcilerMock(t)

		err := rec.setupOneAgentInjection(t.Context(), dk, versionReconciler, connectionInfoReconciler)
		require.NoError(t, err)
	})

	t.Run("injection - ApplicationMonitoring", func(t *testing.T) {
		clt := clientOneAgentInjection()
		rec := createReconciler(clt)
		dk := createDynaKube(testDynakube, testNamespaceDynatrace, oneagent.Spec{
			ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
		})
		rec.istioReconciler = createIstioReconcilerMock(t, dk)

		versionReconciler := createVersionReconcilerMock(t)
		connectionInfoReconciler := createReconcilerMock(t)

		err := rec.setupOneAgentInjection(t.Context(), dk, versionReconciler, connectionInfoReconciler)
		require.NoError(t, err)
	})

	t.Run("injection - CloudNativeFullStack", func(t *testing.T) {
		clt := clientOneAgentInjection()
		rec := createReconciler(clt)
		dk := createDynaKube(testDynakube, testNamespaceDynatrace, oneagent.Spec{
			CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
		})
		rec.istioReconciler = createIstioReconcilerMock(t, dk)

		versionReconciler := createVersionReconcilerMock(t)
		connectionInfoReconciler := createReconcilerMock(t)

		err := rec.setupOneAgentInjection(t.Context(), dk, versionReconciler, connectionInfoReconciler)
		require.NoError(t, err)
	})
}

func TestSetupEnrichmentInjection(t *testing.T) {
	t.Run("no enrichment injection", func(t *testing.T) {
		clt := clientNoInjection()
		rec := createReconciler(clt)
		dk := createDynaKube(testDynakube, testNamespaceDynatrace, oneagent.Spec{
			CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
		})
		dk.Spec.MetadataEnrichment.Enabled = ptr.To(false)

		enrichmentRulesReconciler := createReconcilerMock(t)

		err := rec.setupEnrichmentInjection(t.Context(), dk, enrichmentRulesReconciler)
		require.NoError(t, err)
	})

	t.Run("enrichment injection", func(t *testing.T) {
		clt := clientEnrichmentInjection()
		rec := createReconciler(clt)
		dk := createDynaKube(testDynakube, testNamespaceDynatrace, oneagent.Spec{
			CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
		})
		dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)

		enrichmentRulesReconciler := createReconcilerMock(t)

		err := rec.setupEnrichmentInjection(t.Context(), dk, enrichmentRulesReconciler)
		require.NoError(t, err)
	})
}

func TestGenerateCorrectInitSecret(t *testing.T) {
	ctx := t.Context()
	dkBase := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-dynakube",
			Namespace: "my-dynatrace",
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "url",
			OneAgent: oneagent.Spec{
				ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
			},
		},
	}

	namespaces := []*corev1.Namespace{
		clientInjectedNamespace("ns-1", dkBase.Name),
		clientInjectedNamespace("ns-2", dkBase.Name),
	}

	tokenSecret := clientSecret(dkBase.Name, dkBase.Namespace, map[string][]byte{
		token.APIKey:  []byte("testAPIToken"),
		token.PaaSKey: []byte("testPaasToken"),
	})

	tenantSecret := clientSecret(dkBase.OneAgent().GetTenantSecret(), dkBase.Namespace, map[string][]byte{
		"tenant-token": []byte("token"),
	})

	t.Run("create new secret", func(t *testing.T) {
		dk := dkBase.DeepCopy()

		clt := fake.NewClientWithIndex(
			tokenSecret,
			dk,
			namespaces[0], namespaces[1],
			tenantSecret,
		)

		oneAgentClient := oneagentclientmock.NewClient(t)
		oneAgentClient.EXPECT().GetProcessModuleConfig(anyCtx).Return(&oneagentclient.ProcessModuleConfig{}, nil).Once()

		dtClient := &dynatrace.Client{OneAgent: oneAgentClient}

		r := Reconciler{client: clt, apiReader: clt}

		err := r.generateInitSecret(ctx, dtClient, []corev1.Namespace{*namespaces[0], *namespaces[1]}, dk)
		require.NoError(t, err)

		for _, ns := range namespaces {
			assertSecretFound(t, clt, consts.BootstrapperInitSecretName, ns.Name)
		}

		assertSecretFound(t, clt, bootstrapperconfig.GetSourceConfigSecretName(dk.Name), dk.Namespace)
	})
}

func TestGenerateCorrectCertInitSecret(t *testing.T) {
	ctx := t.Context()
	dkBase := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-dynakube",
			Namespace:   "my-dynatrace",
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "url",
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.RoutingCapability.DisplayName,
				},
			},
			OneAgent: oneagent.Spec{
				ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
			},
		},
	}

	namespaces := []*corev1.Namespace{
		clientInjectedNamespace("ns-1", dkBase.Name),
		clientInjectedNamespace("ns-2", dkBase.Name),
	}

	tokenSecret := clientSecret(dkBase.Name, dkBase.Namespace, map[string][]byte{
		token.APIKey:  []byte(testAPIToken),
		token.PaaSKey: []byte(testPaasToken),
	})

	tenantSecret := clientSecret(dkBase.OneAgent().GetTenantSecret(), dkBase.Namespace, map[string][]byte{
		"tenant-token": []byte("token"),
	})

	autoTLSSecret := clientSecret(dkBase.ActiveGate().GetTLSSecretName(), dkBase.Namespace, map[string][]byte{
		dynakube.TLSCertKey: []byte("certificate"),
	})

	t.Run("create new cert secret and delete it if not needed", func(t *testing.T) {
		dk := dkBase.DeepCopy()

		clt := fake.NewClientWithIndex(
			tokenSecret,
			dk,
			namespaces[0], namespaces[1],
			tenantSecret,
			autoTLSSecret,
		)

		oneAgentClient := oneagentclientmock.NewClient(t)
		oneAgentClient.EXPECT().GetProcessModuleConfig(anyCtx).Return(&oneagentclient.ProcessModuleConfig{}, nil).Once()

		dtClient := &dynatrace.Client{OneAgent: oneAgentClient}

		r := Reconciler{client: clt, apiReader: clt}

		err := r.generateInitSecret(ctx, dtClient, []corev1.Namespace{*namespaces[0], *namespaces[1]}, dk)
		require.NoError(t, err)

		for _, ns := range namespaces {
			assertSecretFound(t, clt, consts.BootstrapperInitCertsSecretName, ns.Name)
		}

		assertSecretFound(t, clt, bootstrapperconfig.GetSourceCertsSecretName(dk.Name), dk.Namespace)

		dk.Annotations[exp.AGAutomaticTLSCertificateKey] = "false"

		err = r.generateInitSecret(ctx, dtClient, []corev1.Namespace{*namespaces[0], *namespaces[1]}, dk)
		require.NoError(t, err)

		for _, ns := range namespaces {
			assertSecretNotFound(t, clt, consts.BootstrapperInitCertsSecretName, ns.Name)
		}

		assertSecretNotFound(t, clt, bootstrapperconfig.GetSourceCertsSecretName(dk.Name), dk.Namespace)
	})
}

func TestGenerateCorrectOTLPCertInitSecret(t *testing.T) {
	ctx := t.Context()
	dkBase := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-dynakube",
			Namespace:   "my-dynatrace",
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "url",
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.RoutingCapability.DisplayName,
				},
			},
			OneAgent: oneagent.Spec{
				ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
			},
			OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
				NamespaceSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						testNamespaceSelectorLabel: testDynakube,
					},
				},
				Signals: otlp.SignalConfiguration{
					Metrics: &otlp.MetricsSignal{},
				},
			},
		},
	}

	namespaces := []*corev1.Namespace{
		clientInjectedNamespace("ns-1", dkBase.Name),
		clientInjectedNamespace("ns-2", dkBase.Name),
	}

	tokenSecret := clientSecret(dkBase.Name, dkBase.Namespace, map[string][]byte{
		token.APIKey:        []byte(testAPIToken),
		token.PaaSKey:       []byte(testPaasToken),
		token.DataIngestKey: []byte(testDataIngestToken),
	})

	tenantSecret := clientSecret(dkBase.OneAgent().GetTenantSecret(), dkBase.Namespace, map[string][]byte{
		"tenant-token": []byte("token"),
	})

	autoTLSSecret := clientSecret(dkBase.ActiveGate().GetTLSSecretName(), dkBase.Namespace, map[string][]byte{
		dynakube.TLSCertKey: []byte("certificate"),
	})

	t.Run("create new cert secret and delete it if not needed", func(t *testing.T) {
		dk := dkBase.DeepCopy()

		clt := fake.NewClientWithIndex(
			tokenSecret,
			dk,
			namespaces[0], namespaces[1],
			tenantSecret,
			autoTLSSecret,
		)

		r := Reconciler{client: clt, apiReader: clt}

		err := r.generateOTLPSecret(ctx, []corev1.Namespace{*namespaces[0], *namespaces[1]}, dk)
		require.NoError(t, err)

		for _, ns := range namespaces {
			assertSecretFound(t, clt, consts.OTLPExporterCertsSecretName, ns.Name)
		}

		assertSecretFound(t, clt, exporterconfig.GetSourceCertsSecretName(dk.Name), dk.Namespace)

		dk.Annotations[exp.AGAutomaticTLSCertificateKey] = "false"

		err = r.generateOTLPSecret(ctx, []corev1.Namespace{*namespaces[0], *namespaces[1]}, dk)
		require.NoError(t, err)

		for _, ns := range namespaces {
			assertSecretNotFound(t, clt, consts.OTLPExporterCertsSecretName, ns.Name)
		}

		assertSecretNotFound(t, clt, exporterconfig.GetSourceCertsSecretName(dk.Name), dk.Namespace)
	})
}

func TestCleanupOneAgentInjection(t *testing.T) {
	ctx := t.Context()
	dkBase := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-dynakube",
			Namespace: "my-dynatrace",
		},
		Spec: dynakube.DynaKubeSpec{},
	}

	t.Run("remove everything", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		namespaces := []*corev1.Namespace{
			clientInjectedNamespace("ns-1", dk.Name),
			clientInjectedNamespace("ns-2", dk.Name),
		}

		setCodeModulesInjectionCreatedCondition(dk.Conditions())

		clt := fake.NewClientWithIndex(
			clientSecret(consts.BootstrapperInitSecretName, namespaces[0].Name, nil),
			clientSecret(consts.BootstrapperInitSecretName, namespaces[1].Name, nil),
			clientSecret(bootstrapperconfig.GetSourceConfigSecretName(dk.Name), dk.Namespace, nil),
			dk,
			namespaces[0], namespaces[1],
		)
		r := Reconciler{client: clt, apiReader: clt}

		r.unmap(ctx, dk)
		r.cleanupInitSecret(ctx, []corev1.Namespace{*namespaces[0], *namespaces[1]}, dk)

		for _, ns := range namespaces {
			assertSecretNotFound(t, clt, consts.BootstrapperInitSecretName, ns.Name)
		}

		assertSecretNotFound(t, clt, bootstrapperconfig.GetSourceConfigSecretName(dk.Name), dk.Namespace)

		assert.Empty(t, dk.Conditions())
	})
}

func TestCleanupOTLPInjection(t *testing.T) {
	ctx := t.Context()
	dkBase := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-dynakube",
			Namespace: "my-dynatrace",
		},
		Spec: dynakube.DynaKubeSpec{},
	}

	t.Run("remove everything", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		namespaces := []*corev1.Namespace{
			clientInjectedNamespace("ns-1", dk.Name),
			clientInjectedNamespace("ns-2", dk.Name),
		}

		setOTLPExporterConfigurationCondition(dk.Conditions())

		clt := fake.NewClientWithIndex(
			clientSecret(consts.OTLPExporterSecretName, namespaces[0].Name, nil),
			clientSecret(consts.OTLPExporterSecretName, namespaces[1].Name, nil),
			clientSecret(bootstrapperconfig.GetSourceConfigSecretName(dk.Name), dk.Namespace, nil),
			dk,
			namespaces[0], namespaces[1],
		)
		r := Reconciler{client: clt, apiReader: clt}

		r.unmap(ctx, dk)
		r.cleanupOTLPSecret(ctx, []corev1.Namespace{*namespaces[0], *namespaces[1]}, dk)

		for _, ns := range namespaces {
			assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, ns.Name)
		}

		assertSecretNotFound(t, clt, exporterconfig.GetSourceConfigSecretName(dk.Name), dk.Namespace)

		assert.Empty(t, dk.Conditions())
	})
}

func createReconciler(clt client.Client) Reconciler {
	return Reconciler{
		client:    clt,
		apiReader: clt,
	}
}

func createDynaKube(dynakubeName string, dynakubeNamespace string, oneAgentSpec oneagent.Spec) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakubeName,
			Namespace: dynakubeNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:      testAPIURL,
			OneAgent:    oneAgentSpec,
			EnableIstio: true,
		},
	}
}

func clientRemoveAppInjection() client.Client {
	return fake.NewClientWithIndex(
		clientInjectedNamespace(testNamespace, testDynakube),
		clientInjectedNamespace(testNamespace2, testDynakube2),
	)
}

func clientNoInjection() client.Client {
	return fake.NewClientWithIndex(
		clientInjectedNamespace(testNamespace, testDynakube),
		clientInjectedNamespace(testNamespace2, testDynakube2),
	)
}

func clientOneAgentInjection() client.Client {
	return fake.NewClientWithIndex(
		clientInjectedNamespace(testNamespace, testDynakube),
		clientInjectedNamespace(testNamespace2, testDynakube2),
		clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
			token.APIKey:  []byte(testAPIToken),
			token.PaaSKey: []byte(testPaasToken),
		}),
	)
}

func clientEnrichmentInjection() client.Client {
	return fake.NewClientWithIndex(
		clientInjectedNamespace(testNamespace, testDynakube),
		clientInjectedNamespace(testNamespace2, testDynakube2),
		clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
			token.APIKey:        []byte(testAPIToken),
			token.PaaSKey:       []byte(testPaasToken),
			token.DataIngestKey: []byte(testDataIngestToken),
		}),
	)
}

func clientInjectedNamespace(namespaceName string, dynakubeName string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "corev1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				dtwebhook.InjectionInstanceLabel: dynakubeName,
			},
		},
	}
}

func clientNotInjectedNamespace(namespaceName string, dynakubeName string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "corev1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				testNamespaceSelectorLabel: dynakubeName,
			},
		},
	}
}

func clientSecret(secretName string, namespaceName string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespaceName,
		},
		Data: data,
	}
}

func assertSecretFound(t *testing.T, clt client.Client, secretName string, secretNamespace string) {
	var secret corev1.Secret
	err := clt.Get(t.Context(), client.ObjectKey{Name: secretName, Namespace: secretNamespace}, &secret)
	require.NoError(t, err, "%s.%s secret not found, error: %s", secretName, secretNamespace, err)
}

func assertSecretNotFound(t *testing.T, clt client.Client, secretName string, secretNamespace string) {
	var secret corev1.Secret
	err := clt.Get(t.Context(), client.ObjectKey{Name: secretName, Namespace: secretNamespace}, &secret)
	require.Error(t, err, "%s.%s secret found, error: %s ", secretName, secretNamespace, err)
	assert.True(t, k8serrors.IsNotFound(err), "%s.%s secret, unexpected error: %s", secretName, secretNamespace, err)
}

func createReconcilerMock(t *testing.T) controllers.Reconciler {
	connectionInfoReconciler := controllermock.NewReconciler(t)
	connectionInfoReconciler.EXPECT().Reconcile(anyCtx).Return(nil)

	return connectionInfoReconciler
}

func createVersionReconcilerMock(t *testing.T) versions.Reconciler {
	versionReconciler := versionmock.NewReconciler(t)
	versionReconciler.EXPECT().ReconcileCodeModules(anyCtx, mock.AnythingOfType("*dynakube.DynaKube")).Return(nil).Once()

	return versionReconciler
}

func createIstioReconcilerMock(t *testing.T, dk *dynakube.DynaKube) istioReconciler {
	rec := newMockIstioReconciler(t)

	rec.EXPECT().ReconcileCodeModules(t.Context(), dk).Return(nil).Once()

	return rec
}
