package oneagentapm

import (
	"context"
	"os"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	utilruntime.Must(dynatracev1alpha1.AddToScheme(scheme.Scheme))
}

const (
	apiURL    = "https://ENVIRONMENTID.live.dynatrace.com/api"
	name      = "oneagent"
	namespace = "dynatrace"
)

func TestReconcileOneAgentAPM(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme,
		&dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL: apiURL,
				OneAgentCodeModule: dynatracev1alpha1.OneAgentCodeModuleSpec{
					Enabled: true,
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Data:       map[string][]byte{utils.DynatracePaasToken: []byte("42")},
		},
	)

	dtClient := &dtclient.MockDynatraceClient{}
	dtClient.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
	dtClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{TenantUUID: "abc123456"}, nil)

	reconciler := &ReconcileOneAgentAPM{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              fakeClient,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtClient),
			UpdatePaaSToken:     true,
		},
	}

	_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}})
	assert.NoError(t, err)

	var result dynatracev1alpha1.DynaKube
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &result))
	assert.Equal(t, namespace, result.GetNamespace())
	assert.Equal(t, name, result.GetName())
	assert.True(t, result.Status.BaseOneAgentStatus.Conditions.IsTrueFor(dynatracev1alpha1.PaaSTokenConditionType))
	assert.True(t, result.Status.BaseOneAgentStatus.Conditions.IsUnknownFor(dynatracev1alpha1.APITokenConditionType))
	assert.Equal(t, utils.GetTokensName(&result), result.Status.Tokens)
	mock.AssertExpectationsForObjects(t, dtClient)
}

func TestReconcileOneAgentAPM_MissingToken(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme,
		&dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL: apiURL,
				OneAgentCodeModule: dynatracev1alpha1.OneAgentCodeModuleSpec{
					Enabled: true,
				},
			},
		})

	dtClient := &dtclient.MockDynatraceClient{}

	reconciler := &ReconcileOneAgentAPM{
		client:    fakeClient,
		apiReader: fakeClient,
		scheme:    scheme.Scheme,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              fakeClient,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtClient),
			UpdatePaaSToken:     true,
		},
	}

	_, err := reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}})
	assert.NotNil(t, err)
	assert.Equal(t, "Secret 'dynatrace:oneagent' not found", err.Error())

	var result dynatracev1alpha1.DynaKube
	assert.NoError(t, fakeClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &result))
	assert.Equal(t, namespace, result.GetNamespace())
	assert.Equal(t, name, result.GetName())
	assert.True(t, result.Status.BaseOneAgentStatus.Conditions.IsFalseFor(dynatracev1alpha1.PaaSTokenConditionType))
	assert.True(t, result.Status.BaseOneAgentStatus.Conditions.IsUnknownFor(dynatracev1alpha1.APITokenConditionType))
	assert.Equal(t, utils.GetTokensName(&result), result.Status.Tokens)
	mock.AssertExpectationsForObjects(t, dtClient)
}
