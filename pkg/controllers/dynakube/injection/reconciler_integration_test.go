// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package injection

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	oneagentclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	oneagentclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Integration tests for the injection reconciler against a real API server. Drives one DynaKube and
// two real namespaces through the namespace-selector scenarios described in ICP-8342: each phase
// updates the DynaKube's OneAgent/MetadataEnrichment/OTLP namespace selectors and reconciles once,
// then asserts on the real Secrets left behind (or cleaned up) in each namespace.

const (
	lifecycleNamespace  = "dynatrace-lifecycle"
	lifecycleDynaKube   = "lifecycle-dk"
	lifecycleAPIURL     = "https://tenant.live.dynatrace.com/api"
	lifecycleMEID       = "KUBERNETES_CLUSTER-lifecycle"
	lifecycleNamespaceA = "shop"
	lifecycleNamespaceB = "checkout"
	lifecycleGroupLabel = "group"
)

func TestReconcileSecretReplicationLifecycle(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)

	integrationtests.CreateNamespace(t, t.Context(), clt, lifecycleNamespace)
	integrationtests.CreateNamespace(t, t.Context(), clt, lifecycleNamespaceA)
	integrationtests.CreateNamespace(t, t.Context(), clt, lifecycleNamespaceB)
	labelNamespace(t, clt, lifecycleNamespaceA, "a")
	labelNamespace(t, clt, lifecycleNamespaceB, "b")

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lifecycleDynaKube,
			Namespace: lifecycleNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: lifecycleAPIURL,
			OneAgent: oneagent.Spec{
				ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
			},
			MetadataEnrichment: metadataenrichment.Spec{
				Enabled: new(bool),
			},
			OTLPExporterConfiguration: &otlp.ExporterConfigurationSpec{
				Signals: otlp.SignalConfiguration{Metrics: &otlp.MetricsSignal{}},
			},
		},
		Status: dynakube.DynaKubeStatus{
			KubernetesClusterMEID: lifecycleMEID,
			APIToken:              dynakube.APITokenStatus{Platform: new(bool)},
		},
	}
	*dk.Spec.MetadataEnrichment.Enabled = true
	*dk.Status.APIToken.Platform = true

	integrationtests.CreateDynakube(t, t.Context(), clt, dk)

	integrationtests.CreateKubernetesObject(t, t.Context(), clt, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: dk.Tokens(), Namespace: dk.Namespace},
		Data: map[string][]byte{
			token.APIKey:        []byte("test-api-token"),
			token.PaaSKey:       []byte("test-paas-token"),
			token.DataIngestKey: []byte("test-ingest-token"),
		},
	})

	oneAgentClient := oneagentclientmock.NewClient(t)
	oneAgentClient.EXPECT().GetConnectionInfo(anyCtx, mock.Anything).Return(oneagentclient.ConnectionInfo{
		TenantUUID:  "test-uuid",
		TenantToken: "test-tenant-token",
		Endpoints:   "https://tenant.live.dynatrace.com:443",
	}, nil)
	oneAgentClient.EXPECT().GetProcessGroupingConfig(anyCtx, lifecycleMEID, "").Return(&oneagentclient.ProcessGroupConfig{}, nil)
	oneAgentClient.EXPECT().GetProcessModuleConfig(anyCtx).Return(&oneagentclient.ProcessModuleConfig{}, nil)
	dtClient := &dynatrace.Client{OneAgent: oneAgentClient}

	rec := NewReconciler(clt, clt)

	reconcile := func(t *testing.T) {
		t.Helper()

		rec.versionReconciler = createVersionReconcilerMock(t)
		rec.istioReconciler = createIstioReconcilerMock(t, dk)
		rec.enrichmentRulesReconciler = createEnrichmentRulesReconcilerMock(t)

		require.NoError(t, rec.Reconcile(t.Context(), dtClient, dk))
	}

	t.Run("no selector replicates everywhere", func(t *testing.T) {
		reconcile(t)

		assertSecretIn(t, clt, consts.BootstrapperInitSecretName, lifecycleNamespaceA, lifecycleNamespaceB)
		assertSecretIn(t, clt, consts.OTLPExporterSecretName, lifecycleNamespaceA, lifecycleNamespaceB)
	})

	t.Run("oneagent selector alone still replicates everywhere", func(t *testing.T) {
		setOneAgentSelector(t, clt, dk, groupSelector("a"))

		reconcile(t)

		// MetadataEnrichment has no selector, so its "everything" match still ORs the init secret everywhere.
		assertSecretIn(t, clt, consts.BootstrapperInitSecretName, lifecycleNamespaceA, lifecycleNamespaceB)
		assertSecretIn(t, clt, consts.OTLPExporterSecretName, lifecycleNamespaceA, lifecycleNamespaceB)
	})

	t.Run("metadata selector alone still replicates everywhere", func(t *testing.T) {
		setOneAgentSelector(t, clt, dk, metav1.LabelSelector{})
		setMetadataSelector(t, clt, dk, groupSelector("a"))

		reconcile(t)

		assertSecretIn(t, clt, consts.BootstrapperInitSecretName, lifecycleNamespaceA, lifecycleNamespaceB)
		assertSecretIn(t, clt, consts.OTLPExporterSecretName, lifecycleNamespaceA, lifecycleNamespaceB)
	})

	t.Run("oneagent and metadata selector scopes init secret, otlp stays everywhere", func(t *testing.T) {
		setOneAgentSelector(t, clt, dk, groupSelector("a"))
		setMetadataSelector(t, clt, dk, groupSelector("a"))

		reconcile(t)

		assertSecretIn(t, clt, consts.BootstrapperInitSecretName, lifecycleNamespaceA)
		assertSecretNotIn(t, clt, consts.BootstrapperInitSecretName, lifecycleNamespaceB)
		assertSecretIn(t, clt, consts.OTLPExporterSecretName, lifecycleNamespaceA, lifecycleNamespaceB)
		// checkout still matches OTLP's unrestricted selector, so it must stay mapped even though it
		// lost the init secret specifically - the divergent-selector case, not a full deselection.
		assertNamespaceMapped(t, clt, dk, lifecycleNamespaceB)
	})

	t.Run("otlp selector scopes otlp secret, oneagent and metadata revert to everywhere", func(t *testing.T) {
		setOneAgentSelector(t, clt, dk, metav1.LabelSelector{})
		setMetadataSelector(t, clt, dk, metav1.LabelSelector{})
		setOTLPSelector(t, clt, dk, groupSelector("a"))

		reconcile(t)

		assertSecretIn(t, clt, consts.BootstrapperInitSecretName, lifecycleNamespaceA, lifecycleNamespaceB)
		assertSecretIn(t, clt, consts.OTLPExporterSecretName, lifecycleNamespaceA)
		assertSecretNotIn(t, clt, consts.OTLPExporterSecretName, lifecycleNamespaceB)
	})

	t.Run("all selectors scope independently", func(t *testing.T) {
		setOneAgentSelector(t, clt, dk, groupSelector("a"))
		setMetadataSelector(t, clt, dk, groupSelector("a"))
		setOTLPSelector(t, clt, dk, groupSelector("b"))

		reconcile(t)

		assertSecretIn(t, clt, consts.BootstrapperInitSecretName, lifecycleNamespaceA)
		assertSecretNotIn(t, clt, consts.BootstrapperInitSecretName, lifecycleNamespaceB)
		assertSecretIn(t, clt, consts.OTLPExporterSecretName, lifecycleNamespaceB)
		assertSecretNotIn(t, clt, consts.OTLPExporterSecretName, lifecycleNamespaceA)
	})

	t.Run("updating a selector to no longer match cleans up its secret in that namespace", func(t *testing.T) {
		// checkout (namespace B) still matches nothing on OneAgent/MetadataEnrichment, and now loses its
		// only remaining match (OTLP) too: it should end up with no secrets and be fully unmapped.
		setOTLPSelector(t, clt, dk, groupSelector("a"))

		reconcile(t)

		assertSecretIn(t, clt, consts.BootstrapperInitSecretName, lifecycleNamespaceA)
		assertSecretIn(t, clt, consts.OTLPExporterSecretName, lifecycleNamespaceA)
		assertSecretNotIn(t, clt, consts.BootstrapperInitSecretName, lifecycleNamespaceB)
		assertSecretNotIn(t, clt, consts.OTLPExporterSecretName, lifecycleNamespaceB)
		assertNamespaceUnmapped(t, clt, lifecycleNamespaceB)
	})
}

func labelNamespace(t *testing.T, clt client.Client, name, group string) {
	t.Helper()

	var ns corev1.Namespace
	require.NoError(t, clt.Get(t.Context(), types.NamespacedName{Name: name}, &ns))

	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}

	ns.Labels[lifecycleGroupLabel] = group
	require.NoError(t, clt.Update(t.Context(), &ns))
}

func groupSelector(group string) metav1.LabelSelector {
	return metav1.LabelSelector{MatchLabels: map[string]string{lifecycleGroupLabel: group}}
}

func setOneAgentSelector(t *testing.T, clt client.Client, dk *dynakube.DynaKube, selector metav1.LabelSelector) {
	t.Helper()

	dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = selector
	require.NoError(t, clt.Update(t.Context(), dk))
}

func setMetadataSelector(t *testing.T, clt client.Client, dk *dynakube.DynaKube, selector metav1.LabelSelector) {
	t.Helper()

	dk.Spec.MetadataEnrichment.NamespaceSelector = selector
	require.NoError(t, clt.Update(t.Context(), dk))
}

func setOTLPSelector(t *testing.T, clt client.Client, dk *dynakube.DynaKube, selector metav1.LabelSelector) {
	t.Helper()

	dk.Spec.OTLPExporterConfiguration.NamespaceSelector = selector
	require.NoError(t, clt.Update(t.Context(), dk))
}

func assertNamespaceMapped(t *testing.T, clt client.Client, dk *dynakube.DynaKube, namespace string) {
	t.Helper()

	var ns corev1.Namespace
	require.NoError(t, clt.Get(t.Context(), types.NamespacedName{Name: namespace}, &ns))
	assert.Equalf(t, dk.Name, ns.Labels[dtwebhook.InjectionInstanceLabel], "%s should still be mapped to %s", namespace, dk.Name)
}

func assertNamespaceUnmapped(t *testing.T, clt client.Client, namespace string) {
	t.Helper()

	var ns corev1.Namespace
	require.NoError(t, clt.Get(t.Context(), types.NamespacedName{Name: namespace}, &ns))
	assert.NotContainsf(t, ns.Labels, dtwebhook.InjectionInstanceLabel, "%s should be fully unmapped", namespace)
}

func assertSecretIn(t *testing.T, clt client.Client, secretName string, namespaces ...string) {
	t.Helper()

	for _, ns := range namespaces {
		var secret corev1.Secret
		err := clt.Get(t.Context(), types.NamespacedName{Name: secretName, Namespace: ns}, &secret)
		require.NoErrorf(t, err, "%s.%s should exist", ns, secretName)
	}
}

func assertSecretNotIn(t *testing.T, clt client.Client, secretName string, namespaces ...string) {
	t.Helper()

	for _, ns := range namespaces {
		var secret corev1.Secret
		err := clt.Get(t.Context(), types.NamespacedName{Name: secretName, Namespace: ns}, &secret)
		require.Errorf(t, err, "%s.%s should not exist", ns, secretName)
	}
}
