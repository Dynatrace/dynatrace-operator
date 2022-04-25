package dynakube

import (
	"context"
	"fmt"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileDynatraceClient_TokenValidation(t *testing.T) {
	namespace := "dynatrace"
	dynaKube := "dynakube"
	base := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dynaKube, Namespace: namespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dynaKube,
		},
	}

	t.Run("No secret", func(t *testing.T) {
		deepCopy := base.DeepCopy()
		c := fake.NewClient()
		dtcMock := &dtclient.MockDynatraceClient{}

		rec := &DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now:                 metav1.Now(),
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), deepCopy)
		assert.Nil(t, dtc)
		assert.True(t, ucr)
		assert.False(t, rec.ValidTokens)
		assert.Nil(t, err)

		cond := meta.FindStatusCondition(deepCopy.Status.Conditions, dynatracev1beta1.PaaSTokenConditionType)
		assert.Nil(t, cond)
		AssertCondition(t, deepCopy, dynatracev1beta1.APITokenConditionType, false, dynatracev1beta1.ReasonTokenSecretNotFound,
			"Secret 'dynatrace:dynakube' not found")

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("PaaS token is empty, API token is missing", func(t *testing.T) {
		dk := base.DeepCopy()
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatracePaasToken: ""}))
		dtcMock := &dtclient.MockDynatraceClient{}

		rec := &DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now:                 metav1.Now(),
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), dk)
		assert.Nil(t, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)
		assert.False(t, rec.ValidTokens)

		cond := meta.FindStatusCondition(dk.Status.Conditions, dynatracev1beta1.PaaSTokenConditionType)
		assert.Nil(t, cond)
		AssertCondition(t, dk, dynatracev1beta1.APITokenConditionType, false, dynatracev1beta1.ReasonTokenMissing,
			"Token apiToken on secret dynatrace:dynakube missing")

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("PaaS token is empty, API token should be used", func(t *testing.T) {
		dk := base.DeepCopy()
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatraceApiToken: "84"}))
		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "84").Return(
			dtclient.TokenScopes{dtclient.TokenScopeDataExport,
				dtclient.TokenScopeInstallerDownload,
			}, nil)

		rec := &DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now:                 metav1.Now(),
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), dk)
		assert.NotNil(t, dtc)
		assert.True(t, ucr)
		assert.Nil(t, err)

		cond := meta.FindStatusCondition(dk.Status.Conditions, dynatracev1beta1.PaaSTokenConditionType)
		assert.Nil(t, cond)
		AssertCondition(t, dk, dynatracev1beta1.APITokenConditionType, true, dynatracev1beta1.ReasonTokenReady,
			"Ready")

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("Unauthorized PaaS token, unexpected error for API token request", func(t *testing.T) {
		dk := base.DeepCopy()
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes(nil), dtclient.ServerError{Code: 401, Message: "Token Authentication failed"})
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes(nil), fmt.Errorf("random error"))

		rec := &DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now:                 metav1.Now(),
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)
		assert.False(t, rec.ValidTokens)

		AssertCondition(t, dk, dynatracev1beta1.PaaSTokenConditionType, false, dynatracev1beta1.ReasonTokenUnauthorized,
			"Token on secret dynatrace:dynakube unauthorized")
		AssertCondition(t, dk, dynatracev1beta1.APITokenConditionType, false, dynatracev1beta1.ReasonTokenError,
			"error when querying token on secret dynatrace:dynakube: random error")

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("PaaS token has wrong scope, API token has leading and trailing space characters", func(t *testing.T) {
		dk := base.DeepCopy()
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: " \t84\n  "}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		rec := &DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now:                 metav1.Now(),
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)

		AssertCondition(t, dk, dynatracev1beta1.PaaSTokenConditionType, false, dynatracev1beta1.ReasonTokenScopeMissing,
			"Token on secret dynatrace:dynakube missing scopes [InstallerDownload]")
		AssertCondition(t, dk, dynatracev1beta1.APITokenConditionType, false, dynatracev1beta1.ReasonTokenUnauthorized,
			"Token on secret dynatrace:dynakube has leading and/or trailing spaces")

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("InstallerDownload permission is sufficient for dynakube with feature-disable-hosts-requests", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Annotations = map[string]string{
			"operator.dynatrace.com/feature-disable-hosts-requests": "true",
		}
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatraceApiToken: "84"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)

		rec := &DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now:                 metav1.Now(),
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)

		AssertCondition(t, dk, dynatracev1beta1.APITokenConditionType, true, dynatracev1beta1.ReasonTokenReady, "Ready")
	})

	t.Run("ApiToken without permissions and PaasToken with InstallerDownload permission is sufficient for dynakube with feature-disable-hosts-requests", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Annotations = map[string]string{
			"operator.dynatrace.com/feature-disable-hosts-requests": "true",
		}
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{}, nil)

		rec := &DynatraceClientReconciler{
			Client: c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now: metav1.Now(),
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)

		AssertCondition(t, dk, dynatracev1beta1.PaaSTokenConditionType, true, dynatracev1beta1.ReasonTokenReady, "Ready")
		AssertCondition(t, dk, dynatracev1beta1.APITokenConditionType, true, dynatracev1beta1.ReasonTokenReady, "Ready")
	})

	t.Run("PaaS and API token are ready", func(t *testing.T) {
		dk := base.DeepCopy()
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		rec := &DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now:                 metav1.Now(),
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)

		AssertCondition(t, dk, dynatracev1beta1.PaaSTokenConditionType, true, dynatracev1beta1.ReasonTokenReady, "Ready")
		AssertCondition(t, dk, dynatracev1beta1.APITokenConditionType, true, dynatracev1beta1.ReasonTokenReady, "Ready")

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("API token has missing scope for automatic kubernetes api monitoring", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureAutomaticKubernetesApiMonitoring: "true",
		}
		dk.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{
			Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName},
		}
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport,
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite,
		}, nil)

		rec := &DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now:                 metav1.Now(),
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)
		assert.False(t, rec.ValidTokens)

		AssertCondition(t, dk, dynatracev1beta1.PaaSTokenConditionType, true, dynatracev1beta1.ReasonTokenReady, "Ready")
		AssertCondition(t, dk, dynatracev1beta1.APITokenConditionType, false, dynatracev1beta1.ReasonTokenScopeMissing,
			"Token on secret dynatrace:dynakube missing scopes [entities.read]")
		mock.AssertExpectationsForObjects(t, dtcMock)
	})
	t.Run("API token has missing scope for metrics ingest", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{
			Capabilities: []dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.MetricsIngestCapability.DisplayName,
			},
		}
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatraceApiToken: "84", dtclient.DynatraceDataIngestToken: "69"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport,
			dtclient.TokenScopeInstallerDownload,
		}, nil)
		dtcMock.On("GetTokenScopes", "69").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		rec := &DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now:                 metav1.Now(),
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)
		assert.False(t, rec.ValidTokens)

		AssertCondition(t, dk, dynatracev1beta1.APITokenConditionType, true, dynatracev1beta1.ReasonTokenReady, "Ready")
		AssertCondition(t, dk, dynatracev1beta1.DataIngestTokenConditionType, false, dynatracev1beta1.ReasonTokenScopeMissing, "Token on secret dynatrace:dynakube missing scopes [metrics.ingest]")
		mock.AssertExpectationsForObjects(t, dtcMock)
	})
}

func TestReconcileDynatraceClient_ProbeRequests(t *testing.T) {
	now := metav1.Now()

	namespace := "dynatrace"
	dkName := "dynakube"
	base := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: dkName,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}
	meta.SetStatusCondition(&base.Status.Conditions, metav1.Condition{
		Type:    dynatracev1beta1.APITokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1beta1.ReasonTokenReady,
		Message: "Ready",
	})
	meta.SetStatusCondition(&base.Status.Conditions, metav1.Condition{
		Type:    dynatracev1beta1.PaaSTokenConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1beta1.ReasonTokenReady,
		Message: "Ready",
	})

	c := fake.NewClient(NewSecret(dkName, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}))

	t.Run("No request if last probe was recent", func(t *testing.T) {
		lastAPIProbe := metav1.NewTime(now.Add(-3 * time.Minute))
		lastPaaSProbe := metav1.NewTime(now.Add(-3 * time.Minute))

		dk := base.DeepCopy()
		dk.Status.LastAPITokenProbeTimestamp = &lastAPIProbe
		dk.Status.LastPaaSTokenProbeTimestamp = &lastPaaSProbe

		dtcMock := &dtclient.MockDynatraceClient{}

		rec := &DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now:                 now,
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.False(t, ucr)
		assert.NoError(t, err)
		if assert.NotNil(t, dk.Status.LastAPITokenProbeTimestamp) {
			assert.Equal(t, *dk.Status.LastAPITokenProbeTimestamp, lastAPIProbe)
		}
		if assert.NotNil(t, dk.Status.LastPaaSTokenProbeTimestamp) {
			assert.Equal(t, *dk.Status.LastPaaSTokenProbeTimestamp, lastPaaSProbe)
		}
		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("Make request if last probe was not recent", func(t *testing.T) {
		lastAPIProbe := metav1.NewTime(now.Add(-10 * time.Minute))
		lastPaaSProbe := metav1.NewTime(now.Add(-10 * time.Minute))

		dk := base.DeepCopy()
		dk.Status.LastAPITokenProbeTimestamp = &lastAPIProbe
		dk.Status.LastPaaSTokenProbeTimestamp = &lastPaaSProbe

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		rec := &DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: StaticDynatraceClient(dtcMock),
			Now:                 now,
		}

		dtc, ucr, err := rec.Reconcile(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.True(t, ucr)
		assert.NoError(t, err)
		if assert.NotNil(t, dk.Status.LastAPITokenProbeTimestamp) {
			assert.Equal(t, *dk.Status.LastAPITokenProbeTimestamp, now)
		}
		if assert.NotNil(t, dk.Status.LastPaaSTokenProbeTimestamp) {
			assert.Equal(t, *dk.Status.LastPaaSTokenProbeTimestamp, now)
		}
		mock.AssertExpectationsForObjects(t, dtcMock)
	})
}

func AssertCondition(t *testing.T, dk *dynatracev1beta1.DynaKube, ct string, status bool, reason string, message string) {
	t.Helper()
	s := metav1.ConditionFalse
	if status {
		s = metav1.ConditionTrue
	}

	cond := meta.FindStatusCondition(dk.Status.Conditions, ct)
	require.NotNil(t, cond)
	assert.Equal(t, s, cond.Status)
	assert.Equal(t, reason, cond.Reason)
	assert.Equal(t, message, cond.Message)
}

func NewSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}
