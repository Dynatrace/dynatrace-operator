package dynatraceclient

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
		fakeClient := fake.NewClient()
		dtcMock := &dtclient.MockDynatraceClient{}

		rec := &Factory{
			client:              fakeClient,
			dynatraceClientFunc: StaticDynatraceClient(dtcMock),
		}

		result, err := rec.Create(context.TODO(), deepCopy)
		assert.Nil(t, result)
		assert.NotNil(t, err)

		cond := meta.FindStatusCondition(deepCopy.Status.Conditions, dynatracev1beta1.TokenConditionType)
		assert.Nil(t, cond)

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("PaaS token is empty, API token is missing", func(t *testing.T) {
		dk := base.DeepCopy()
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatracePaasToken: ""}))
		dtcMock := &dtclient.MockDynatraceClient{}

		rec := &Factory{
			client:              c,
			dynatraceClientFunc: StaticDynatraceClient(dtcMock),
		}

		result, err := rec.Create(context.TODO(), dk)
		assert.Nil(t, result)
		assert.Error(t, err)

		cond := meta.FindStatusCondition(dk.Status.Conditions, dynatracev1beta1.PaaSTokenConditionType)
		assert.Nil(t, cond)

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("PaaS token is empty, API token should be used", func(t *testing.T) {
		dk := base.DeepCopy()
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatraceApiToken: "84"}))
		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "84").Return(
			dtclient.TokenScopes{dtclient.TokenScopeDataExport,
				dtclient.TokenScopeInstallerDownload,
				dtclient.TokenScopeActiveGateTokenCreate,
			}, nil)
		rec := &Factory{
			client:              c,
			dynatraceClientFunc: StaticDynatraceClient(dtcMock),
		}

		dtc, err := rec.Create(context.TODO(), dk)
		assert.NotNil(t, dtc)
		assert.Nil(t, err)

		cond := meta.FindStatusCondition(dk.Status.Conditions, dynatracev1beta1.PaaSTokenConditionType)
		assert.Nil(t, cond)

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("Unauthorized PaaS token, unexpected error for API token request", func(t *testing.T) {
		dk := base.DeepCopy()
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes(nil), dtclient.ServerError{Code: 401, Message: "Token Authentication failed"})
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes(nil), fmt.Errorf("random error"))

		rec := &Factory{
			client:              c,
			dynatraceClientFunc: StaticDynatraceClient(dtcMock),
		}

		result, err := rec.Create(context.TODO(), dk)
		assert.Nil(t, result)
		assert.Error(t, err)

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("API token has leading and trailing space characters", func(t *testing.T) {
		dk := base.DeepCopy()
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: " \t84\n  "}))

		rec := &Factory{
			client: c,
		}

		dtc, err := rec.Create(context.TODO(), dk)
		assert.Nil(t, dtc)
		assert.Error(t, err)
	})

	t.Run("InstallerDownload permission is sufficient for dynakube with feature-disable-hosts-requests", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Annotations = map[string]string{
			"feature.dynatrace.com/disable-hosts-requests": "true",
		}
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatraceApiToken: "84"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil)

		rec := &Factory{
			client:              c,
			dynatraceClientFunc: StaticDynatraceClient(dtcMock),
		}

		dtc, err := rec.Create(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.NoError(t, err)
	})

	t.Run("ApiToken without permissions and PaasToken with InstallerDownload permission is sufficient for dynakube with feature-disable-hosts-requests", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Annotations = map[string]string{
			"feature.dynatrace.com/disable-hosts-requests": "true",
		}
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeActiveGateTokenCreate}, nil)

		rec := &Factory{
			client:              c,
			dynatraceClientFunc: StaticDynatraceClient(dtcMock),
		}

		dtc, err := rec.Create(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.NoError(t, err)
	})

	t.Run("PaaS and API token are ready", func(t *testing.T) {
		dk := base.DeepCopy()
		c := fake.NewClient(NewSecret(dynaKube, namespace, map[string]string{dtclient.DynatracePaasToken: "42", dtclient.DynatraceApiToken: "84"}))

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport,
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil)

		rec := &Factory{
			client:              c,
			dynatraceClientFunc: StaticDynatraceClient(dtcMock),
		}

		dtc, err := rec.Create(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.NoError(t, err)

		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("API token has missing scope for automatic kubernetes api monitoring", func(t *testing.T) {
		dk := base.DeepCopy()
		dk.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring: "true",
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
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil)

		rec := &Factory{
			client:              c,
			dynatraceClientFunc: StaticDynatraceClient(dtcMock),
		}

		result, err := rec.Create(context.TODO(), dk)
		assert.Nil(t, result)
		assert.Error(t, err)

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
			dtclient.TokenScopeActiveGateTokenCreate,
		}, nil)
		dtcMock.On("GetTokenScopes", "69").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		rec := &Factory{
			client:              c,
			dynatraceClientFunc: StaticDynatraceClient(dtcMock),
		}

		result, err := rec.Create(context.TODO(), dk)
		assert.Nil(t, result)
		assert.Error(t, err)

		mock.AssertExpectationsForObjects(t, dtcMock)
	})
}

func TestReconcileDynatraceClient_ProbeRequests(t *testing.T) {
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
		lastAPIProbe := metav1.NewTime(metav1.Now().Add(-3 * time.Minute))

		dk := base.DeepCopy()
		dk.Status.LastAPITokenProbeTimestamp = &lastAPIProbe

		dtcMock := &dtclient.MockDynatraceClient{}

		rec := &Factory{
			client:              c,
			dynatraceClientFunc: StaticDynatraceClient(dtcMock),
		}

		dtc, err := rec.Create(context.TODO(), dk)
		assert.NotNil(t, dtc)
		assert.NoError(t, err)
		if assert.NotNil(t, dk.Status.LastAPITokenProbeTimestamp) {
			assert.Equal(t, *dk.Status.LastAPITokenProbeTimestamp, lastAPIProbe)
		}
		mock.AssertExpectationsForObjects(t, dtcMock)
	})

	t.Run("Make request if last probe was not recent", func(t *testing.T) {
		lastAPIProbe := metav1.NewTime(metav1.Now().Add(-10 * time.Minute))

		dk := base.DeepCopy()
		dk.Status.LastAPITokenProbeTimestamp = &lastAPIProbe

		dtcMock := &dtclient.MockDynatraceClient{}
		dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		rec := &Factory{
			client:              c,
			dynatraceClientFunc: StaticDynatraceClient(dtcMock),
		}

		dtc, err := rec.Create(context.TODO(), dk)
		assert.Equal(t, dtcMock, dtc)
		assert.NoError(t, err)
		if assert.NotNil(t, dk.Status.LastAPITokenProbeTimestamp) {
			assert.NotEqual(t, lastAPIProbe, *dk.Status.LastAPITokenProbeTimestamp)
		}
		mock.AssertExpectationsForObjects(t, dtcMock)
	})
}

func NewSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}
