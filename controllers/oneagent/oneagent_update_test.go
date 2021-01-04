package oneagent

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewerVersion(t *testing.T) {
	assert.True(t, isDesiredNewer("1.200.1.12345", "1.201.1.12345", consoleLogger))
	assert.True(t, isDesiredNewer("1.200.1.12345", "2.200.1.12345", consoleLogger))
	assert.True(t, isDesiredNewer("1.200.1.12345", "1.200.2.12345", consoleLogger))
	assert.True(t, isDesiredNewer("1.200.1.12345", "1.200.1.123456", consoleLogger))
}

func TestBackportVersion(t *testing.T) {
	assert.False(t, isDesiredNewer("1.202.1.12345", "1.201.1.12345", consoleLogger))
	assert.False(t, isDesiredNewer("1.201.2.12345", "1.201.1.12345", consoleLogger))
	assert.False(t, isDesiredNewer("1.201.1.12345", "1.201.1.12344", consoleLogger))
	assert.False(t, isDesiredNewer("2.201.1.12345", "1.201.1.12345", consoleLogger))
}

func TestSameVersion(t *testing.T) {
	assert.False(t, isDesiredNewer("1.202.1.12345", "1.202.1.12345", consoleLogger))
	assert.False(t, isDesiredNewer("2.202.1.12345", "2.202.1.12345", consoleLogger))
	assert.False(t, isDesiredNewer("1.202.2.12345", "1.202.2.12345", consoleLogger))
	assert.False(t, isDesiredNewer("1.202.1.1", "1.202.1.1", consoleLogger))
}

func TestReconcile_InstallerDowngrade(t *testing.T) {
	var wait uint16 = 5

	namespace := "dynatrace"
	oaName := "oneagent"
	dynakube := dynatracev1alpha1.DynaKube{
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: oaName,
			OneAgent: dynatracev1alpha1.OneAgentSpec{
				Enabled: true,
				WaitReadySeconds: &wait,
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			OneAgentStatus: dynatracev1alpha1.OneAgentStatus{
				Version: "1.206.0.20200101-000000",
			},
		},
	}

	labels := map[string]string{"dynatrace": "oneagent", "oneagent": oaName}

	c := fake.NewFakeClientWithScheme(scheme.Scheme,
		&dynakube,
		NewSecret(oaName, namespace, map[string]string{utils.DynatracePaasToken: "42", utils.DynatraceApiToken: "84"}),
		&corev1.Pod{ // To be untouched.
			ObjectMeta: metav1.ObjectMeta{Name: "future-pod", Namespace: "dynatrace", Labels: labels},
			Spec:       corev1.PodSpec{},
			Status:     corev1.PodStatus{HostIP: "1.2.3.3"},
		},
		&corev1.Pod{ // To be untouched.
			ObjectMeta: metav1.ObjectMeta{Name: "current-pod", Namespace: "dynatrace", Labels: labels},
			Spec:       corev1.PodSpec{},
			Status:     corev1.PodStatus{HostIP: "1.2.3.4"},
		},
		&corev1.Pod{ // To be deleted.
			ObjectMeta: metav1.ObjectMeta{Name: "past-pod", Namespace: "dynatrace", Labels: labels},
			Spec:       corev1.PodSpec{},
			Status:     corev1.PodStatus{HostIP: "1.2.3.5"},
		},
		sampleKubeSystemNS)

	dtcMock := &dtclient.MockDynatraceClient{}
	dtcMock.On("GetLatestAgentVersion", dtclient.OsUnix, dtclient.InstallerTypeDefault).Return("1.202.0.20190101-000000", nil)
	dtcMock.On("GetAgentVersionForIP", "1.2.3.3").Return("1.203.0.20190101-000000", nil)
	dtcMock.On("GetAgentVersionForIP", "1.2.3.4").Return("1.202.0.20190101-000000", nil)
	dtcMock.On("GetAgentVersionForIP", "1.2.3.5").Return("1.201.0.20190101-000000", nil)
	dtcMock.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{utils.DynatracePaasToken}, nil)
	dtcMock.On("GetTokenScopes", "84").Return(dtclient.TokenScopes{utils.DynatraceApiToken}, nil)

	r := &ReconcileOneAgent{
		client:    c,
		apiReader: c,
		scheme:    scheme.Scheme,
		logger:    consoleLogger,
		dtcReconciler: &utils.DynatraceClientReconciler{
			Client:              c,
			DynatraceClientFunc: utils.StaticDynatraceClient(dtcMock),
			UpdatePaaSToken:     true,
			UpdateAPIToken:      true,
		},
	}

	// Fails because the Pod didn't get recreated. Ignore since that isn't what we're checking on this test.
	r.reconcileVersionInstaller(consoleLogger, &dynakube, dtcMock)

	// These Pods should not be restarted, so we should be able to query that the Pod is still there and get no errors.
	assert.NoError(t, c.Get(context.TODO(), types.NamespacedName{Name: "future-pod", Namespace: "dynatrace"}, &corev1.Pod{}))
	assert.NoError(t, c.Get(context.TODO(), types.NamespacedName{Name: "current-pod", Namespace: "dynatrace"}, &corev1.Pod{}))

	// Outdated Pod should be deleted.
	assert.Error(t, c.Get(context.TODO(), types.NamespacedName{Name: "past-pod", Namespace: "dynatrace"}, &corev1.Pod{}))
}
