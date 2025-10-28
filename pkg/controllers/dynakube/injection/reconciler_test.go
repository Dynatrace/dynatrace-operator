package injection

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	versions "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/otlp/exporterconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	versionmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	istio2 "istio.io/api/networking/v1beta1"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	fakeistio "istio.io/client-go/pkg/clientset/versioned/fake"
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

func TestReconciler(t *testing.T) {
	t.Run("add injection", func(t *testing.T) {
		expectedOneAgentConnectionInfo := dtclient.OneAgentConnectionInfo{
			ConnectionInfo: dtclient.ConnectionInfo{
				TenantUUID:  testUUID,
				TenantToken: testTenantToken,
				Endpoints:   testCommunicationEndpoint,
			},
			CommunicationHosts: []dtclient.CommunicationHost{
				{
					Protocol: "https",
					Host:     "tenant.dev.dynatracelabs.com",
					Port:     443,
				},
				{
					Protocol: "https",
					Host:     "1.2.3.4",
					Port:     443,
				},
			},
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
				OTLPExporterConfiguration: &otlp.Spec{
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
		conditions.SetOptionalScopeAvailable(dk.Conditions(), dtclient.ConditionTypeAPITokenSettingsRead, "available")
		clt := fake.NewClientWithIndex(
			clientNotInjectedNamespace(testNamespace, testDynakube),
			clientNotInjectedNamespace(testNamespace2, testDynakube2),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.APIToken:        []byte(testAPIToken),
				dtclient.PaasToken:       []byte(testPaasToken),
				dtclient.DataIngestToken: []byte(testDataIngestToken),
			}),
			dk,
		)
		dtClient := dtclientmock.NewClient(t)
		dtClient.On("GetLatestAgentVersion", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("", nil)
		dtClient.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(expectedOneAgentConnectionInfo, nil)
		dtClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).Return(&dtclient.ProcessModuleConfig{}, nil)
		dtClient.On("GetRulesSettings", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(dtclient.GetRulesSettingsResponse{}, nil)

		istioClient := newIstioTestingClient(fakeistio.NewSimpleClientset(), dk)

		rec := NewReconciler(clt, clt, dtClient, istioClient, dk)
		fakeReconciler := createReconcilerMock(t)
		rec.(*Reconciler).k8sEntityReconciler = fakeReconciler

		err := rec.Reconcile(context.Background())
		require.NoError(t, err)

		assertSecretFound(t, clt, dk.OneAgent().GetTenantSecret(), dk.Namespace)
		assertSecretFound(t, clt, consts.BootstrapperInitSecretName, testNamespace)
		assertSecretNotFound(t, clt, consts.BootstrapperInitSecretName, testNamespace2)

		assertSecretFound(t, clt, consts.OTLPExporterSecretName, testNamespace)
		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, testNamespace2)

		_, err = istioClient.GetServiceEntry(context.Background(), istio.BuildNameForIPServiceEntry(dk.GetName(), istio.OneAgentComponent))
		require.NoError(t, err)
		_, err = istioClient.GetServiceEntry(context.Background(), istio.BuildNameForFQDNServiceEntry(dk.GetName(), istio.OneAgentComponent))
		require.NoError(t, err)

		_, err = istioClient.GetVirtualService(context.Background(), istio.BuildNameForIPServiceEntry(dk.GetName(), istio.OneAgentComponent))
		require.NoError(t, err)
		_, err = istioClient.GetVirtualService(context.Background(), istio.BuildNameForFQDNServiceEntry(dk.GetName(), istio.OneAgentComponent))
		require.NoError(t, err)
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
				dtclient.APIToken:  []byte(testAPIToken),
				dtclient.PaasToken: []byte(testPaasToken),
			}),
			dk,
		)
		dtClient := dtclientmock.NewClient(t)
		istioClient := setupIstioClientWithObjects(dk)

		rec := NewReconciler(clt, clt, dtClient, istioClient, dk)
		fakeReconciler := createUncalledReconcilerMock(t)
		rec.(*Reconciler).k8sEntityReconciler = fakeReconciler

		err := rec.Reconcile(context.Background())
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

		obj, err := istioClient.GetServiceEntry(context.Background(), istio.BuildNameForIPServiceEntry(dk.GetName(), istio.OneAgentComponent))
		require.NoError(t, err)
		assert.Nil(t, obj)
		obj, err = istioClient.GetServiceEntry(context.Background(), istio.BuildNameForFQDNServiceEntry(dk.GetName(), istio.OneAgentComponent))
		require.NoError(t, err)
		assert.Nil(t, obj)

		virtualService, err := istioClient.GetVirtualService(context.Background(), istio.BuildNameForFQDNServiceEntry(dk.GetName(), istio.OneAgentComponent))
		require.NoError(t, err)
		assert.Nil(t, virtualService)

		istioClient.Owner.SetNamespace(testNamespace2)
		obj, err = istioClient.GetServiceEntry(context.Background(), istio.BuildNameForIPServiceEntry(dk.GetName(), istio.OneAgentComponent))
		require.NoError(t, err)
		assert.NotNil(t, obj)
		obj, err = istioClient.GetServiceEntry(context.Background(), istio.BuildNameForIPServiceEntry(dk.GetName(), istio.OneAgentComponent))
		require.NoError(t, err)
		assert.NotNil(t, obj)

		virtualService, err = istioClient.GetVirtualService(context.Background(), istio.BuildNameForFQDNServiceEntry(dk.GetName(), istio.OneAgentComponent))
		require.NoError(t, err)
		assert.NotNil(t, virtualService)
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

		istioClient := newIstioTestingClient(fakeistio.NewSimpleClientset(), dk)
		fakeReconciler := createUncalledReconcilerMock(t)
		fakeVersionReconciler := createVersionReconcilerMock(t)

		rec := NewReconciler(boomClient, boomClient, nil, istioClient, dk).(*Reconciler)
		rec.connectionInfoReconciler = fakeReconciler
		rec.versionReconciler = fakeVersionReconciler
		rec.k8sEntityReconciler = fakeReconciler

		err := rec.Reconcile(context.Background())
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), codeModulesInjectionConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func TestRemoveAppInjection(t *testing.T) {
	clt := clientRemoveAppInjection()
	rec := createReconciler(clt, testDynakube, testNamespaceDynatrace, oneagent.Spec{
		CloudNativeFullStack: nil,
	})
	rec.versionReconciler = createVersionReconcilerMock(t)
	rec.connectionInfoReconciler = createUncalledReconcilerMock(t)
	rec.enrichmentRulesReconciler = createUncalledReconcilerMock(t)
	rec.k8sEntityReconciler = createUncalledReconcilerMock(t)

	setCodeModulesInjectionCreatedCondition(rec.dk.Conditions())
	setMetadataEnrichmentCreatedCondition(rec.dk.Conditions())

	err := rec.Reconcile(context.Background())
	require.NoError(t, err)

	var namespace corev1.Namespace
	err = clt.Get(context.Background(), client.ObjectKey{Name: testNamespace, Namespace: ""}, &namespace)
	require.NoError(t, err)
	assert.Nil(t, namespace.Labels)
	require.NotNil(t, namespace.Annotations)
	assert.Equal(t, "true", namespace.Annotations[mapper.UpdatedViaDynakubeAnnotation])

	err = clt.Get(context.Background(), client.ObjectKey{Name: testNamespace2, Namespace: ""}, &namespace)
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
		rec := createReconciler(clt, testDynakube, testNamespaceDynatrace, oneagent.Spec{
			ClassicFullStack: &oneagent.HostInjectSpec{},
		})
		rec.versionReconciler = createVersionReconcilerMock(t)
		rec.connectionInfoReconciler = createReconcilerMock(t)
		rec.k8sEntityReconciler = createUncalledReconcilerMock(t)

		err := rec.setupOneAgentInjection(context.Background())
		require.NoError(t, err)
	})

	t.Run("no injection - HostMonitoring", func(t *testing.T) {
		clt := clientNoInjection()
		rec := createReconciler(clt, testDynakube, testNamespaceDynatrace, oneagent.Spec{
			HostMonitoring: &oneagent.HostInjectSpec{},
		})
		rec.versionReconciler = createVersionReconcilerMock(t)
		rec.connectionInfoReconciler = createReconcilerMock(t)
		rec.k8sEntityReconciler = createUncalledReconcilerMock(t)

		err := rec.setupOneAgentInjection(context.Background())
		require.NoError(t, err)
	})

	t.Run("injection - ApplicationMonitoring", func(t *testing.T) {
		clt := clientOneAgentInjection()
		rec := createReconciler(clt, testDynakube, testNamespaceDynatrace, oneagent.Spec{
			ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
		})
		rec.versionReconciler = createVersionReconcilerMock(t)
		rec.connectionInfoReconciler = createReconcilerMock(t)
		rec.k8sEntityReconciler = createUncalledReconcilerMock(t)

		err := rec.setupOneAgentInjection(context.Background())
		require.NoError(t, err)
	})

	t.Run("injection - CloudNativeFullStack", func(t *testing.T) {
		clt := clientOneAgentInjection()
		rec := createReconciler(clt, testDynakube, testNamespaceDynatrace, oneagent.Spec{
			CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
		})
		rec.versionReconciler = createVersionReconcilerMock(t)
		rec.connectionInfoReconciler = createReconcilerMock(t)
		rec.k8sEntityReconciler = createUncalledReconcilerMock(t)

		err := rec.setupOneAgentInjection(context.Background())
		require.NoError(t, err)
	})
}

func TestSetupEnrichmentInjection(t *testing.T) {
	t.Run("no enrichment injection", func(t *testing.T) {
		clt := clientNoInjection()
		rec := createReconciler(clt, testDynakube, testNamespaceDynatrace, oneagent.Spec{
			CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
		})
		rec.enrichmentRulesReconciler = createUncalledReconcilerMock(t)
		rec.k8sEntityReconciler = createUncalledReconcilerMock(t)
		rec.dk.Spec.MetadataEnrichment.Enabled = ptr.To(false)

		err := rec.setupEnrichmentInjection(context.Background())
		require.NoError(t, err)
	})

	t.Run("enrichment injection", func(t *testing.T) {
		clt := clientEnrichmentInjection()
		rec := createReconciler(clt, testDynakube, testNamespaceDynatrace, oneagent.Spec{
			CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
		})
		rec.enrichmentRulesReconciler = createReconcilerMock(t)
		rec.k8sEntityReconciler = createReconcilerMock(t)
		rec.dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)

		err := rec.setupEnrichmentInjection(context.Background())
		require.NoError(t, err)
	})
}

func TestGenerateCorrectInitSecret(t *testing.T) {
	ctx := context.Background()
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
		dtclient.APIToken:  []byte("testAPIToken"),
		dtclient.PaasToken: []byte("testPaasToken"),
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

		dtClient := dtclientmock.NewClient(t)
		dtClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).Return(&dtclient.ProcessModuleConfig{}, nil)

		r := Reconciler{client: clt, apiReader: clt, dk: dk, dynatraceClient: dtClient}

		err := r.generateInitSecret(ctx, []corev1.Namespace{*namespaces[0], *namespaces[1]})
		require.NoError(t, err)

		for _, ns := range namespaces {
			assertSecretFound(t, clt, consts.BootstrapperInitSecretName, ns.Name)
		}

		assertSecretFound(t, clt, bootstrapperconfig.GetSourceConfigSecretName(dk.Name), dk.Namespace)
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
		r := Reconciler{client: clt, apiReader: clt, dk: dk}

		r.unmap(ctx)
		r.cleanupInitSecret(ctx, []corev1.Namespace{*namespaces[0], *namespaces[1]})

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
		r := Reconciler{client: clt, apiReader: clt, dk: dk}

		r.unmap(ctx)
		r.cleanupOTLPSecret(ctx, []corev1.Namespace{*namespaces[0], *namespaces[1]})

		for _, ns := range namespaces {
			assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, ns.Name)
		}

		assertSecretNotFound(t, clt, exporterconfig.GetSourceConfigSecretName(dk.Name), dk.Namespace)

		assert.Empty(t, dk.Conditions())
	})
}

func newIstioTestingClient(fakeClient *fakeistio.Clientset, dk *dynakube.DynaKube) *istio.Client {
	return &istio.Client{
		IstioClientset: fakeClient,
		Owner:          dk,
	}
}

func createReconciler(clt client.Client, dynakubeName string, dynakubeNamespace string, oneAgentSpec oneagent.Spec) Reconciler {
	return Reconciler{
		client:    clt,
		apiReader: clt,
		dk: &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dynakubeName,
				Namespace: dynakubeNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL:      testAPIURL,
				OneAgent:    oneAgentSpec,
				EnableIstio: true,
			},
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
			dtclient.APIToken:  []byte(testAPIToken),
			dtclient.PaasToken: []byte(testPaasToken),
		}),
	)
}

func clientEnrichmentInjection() client.Client {
	return fake.NewClientWithIndex(
		clientInjectedNamespace(testNamespace, testDynakube),
		clientInjectedNamespace(testNamespace2, testDynakube2),
		clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
			dtclient.APIToken:        []byte(testAPIToken),
			dtclient.PaasToken:       []byte(testPaasToken),
			dtclient.DataIngestToken: []byte(testDataIngestToken),
		}),
	)
}

func setupIstioClientWithObjects(dk *dynakube.DynaKube) *istio.Client {
	return newIstioTestingClient(fakeistio.NewSimpleClientset(
		clientServiceEntry(istio.BuildNameForIPServiceEntry(dk.Name, istio.OneAgentComponent), testNamespace),
		clientServiceEntry(istio.BuildNameForFQDNServiceEntry(dk.Name, istio.OneAgentComponent), testNamespace),
		clientServiceEntry(istio.BuildNameForIPServiceEntry(dk.Name, istio.OneAgentComponent), testNamespace2),
		clientServiceEntry(istio.BuildNameForFQDNServiceEntry(dk.Name, istio.OneAgentComponent), testNamespace2),

		clientVirtualService(istio.BuildNameForFQDNServiceEntry(dk.Name, istio.OneAgentComponent), testNamespace),
		clientVirtualService(istio.BuildNameForFQDNServiceEntry(dk.Name, istio.OneAgentComponent), testNamespace2),
	), dk)
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

func clientServiceEntry(name string, namespaceName string) *istiov1beta1.ServiceEntry {
	return &istiov1beta1.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
		Spec: istio2.ServiceEntry{},
	}
}

func clientVirtualService(name string, namespaceName string) *istiov1beta1.VirtualService {
	return &istiov1beta1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
		Spec: istio2.VirtualService{},
	}
}

func assertSecretFound(t *testing.T, clt client.Client, secretName string, secretNamespace string) {
	var secret corev1.Secret
	err := clt.Get(context.Background(), client.ObjectKey{Name: secretName, Namespace: secretNamespace}, &secret)
	require.NoError(t, err, "%s.%s secret not found, error: %s", secretName, secretNamespace, err)
}

func assertSecretNotFound(t *testing.T, clt client.Client, secretName string, secretNamespace string) {
	var secret corev1.Secret
	err := clt.Get(context.Background(), client.ObjectKey{Name: secretName, Namespace: secretNamespace}, &secret)
	require.Error(t, err, "%s.%s secret found, error: %s ", secretName, secretNamespace, err)
	assert.True(t, k8serrors.IsNotFound(err), "%s.%s secret, unexpected error: %s", secretName, secretNamespace, err)
}

func createReconcilerMock(t *testing.T) controllers.Reconciler {
	connectionInfoReconciler := controllermock.NewReconciler(t)
	connectionInfoReconciler.On("Reconcile",
		mock.AnythingOfType("context.backgroundCtx")).Return(nil)

	return connectionInfoReconciler
}

func createUncalledReconcilerMock(t *testing.T) controllers.Reconciler {
	connectionInfoReconciler := controllermock.NewReconciler(t)
	connectionInfoReconciler.On("Reconcile",
		mock.AnythingOfType("context.backgroundCtx")).Return(nil).Maybe()

	return connectionInfoReconciler
}

func createVersionReconcilerMock(t *testing.T) versions.Reconciler {
	versionReconciler := versionmock.NewReconciler(t)
	versionReconciler.On("ReconcileCodeModules",
		mock.AnythingOfType("context.backgroundCtx"),
		mock.AnythingOfType("*dynakube.DynaKube")).Return(nil).Once()

	return versionReconciler
}
