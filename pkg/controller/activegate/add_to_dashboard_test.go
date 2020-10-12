package activegate

import (
	"context"
	"fmt"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/dao"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"net/http/httptest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"
)

func TestAddToDashboard(t *testing.T) {
	// Setup
	tokenValue := []byte("super-secret-bearer-token")
	server := httptest.NewServer(func() http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			_, _ = writer.Write(make([]byte, 0))
		}
	}())
	r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
	assert.NoError(t, err)

	instance.Spec.KubernetesAPIEndpoint = server.URL

	err = r.client.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "dynatrace",
			Name:      "dynatrace-activegate-secret",
		},
		Data: map[string][]byte{
			"token": tokenValue,
		},
	})
	assert.NoError(t, err)

	err = r.client.Create(context.TODO(), &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "dynatrace",
			Name:      "dynatrace-activegate",
		},
		Secrets: []corev1.ObjectReference{
			{Name: "dynatrace-activegate-secret"},
		},
	})
	assert.NoError(t, err)

	serviceAccount, err := dao.FindServiceAccount(r.client)
	assert.NoError(t, err)
	assert.NotNil(t, serviceAccount)
	assert.Equal(t, 1, len(serviceAccount.Secrets))

	tokenName := serviceAccount.Secrets[0].Name
	tokenSecret, err := dao.FindBearerTokenSecret(r.client, tokenName)
	assert.NoError(t, err)
	assert.NotNil(t, tokenSecret)
	assert.Equal(t, 1, len(tokenSecret.Data))

	port := strings.Split(server.URL, ":")[2]
	label := "dynatrace-activegate-127-0-0-1-" + port

	t.Run("AddToDashboard", func(t *testing.T) {
		r.dtcBuildFunc = func(rtc client.Client, instance *dynatracev1alpha1.ActiveGate, secret *corev1.Secret) (dtclient.Client, error) {
			mockClient := &dtclient.MockDynatraceClient{}
			mockClient.
				On("AddToDashboard", label, server.URL, string(tokenValue)).
				Return(label, nil)

			return mockClient, nil
		}

		id, err := r.addToDashboard(tokenSecret, instance)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Equal(t, label, id)
	})
	t.Run("AddToDashboard error from api", func(t *testing.T) {
		r.dtcBuildFunc = func(rtc client.Client, instance *dynatracev1alpha1.ActiveGate, secret *corev1.Secret) (dtclient.Client, error) {
			mockClient := &dtclient.MockDynatraceClient{}
			mockClient.
				On("AddToDashboard", label, server.URL, string(tokenValue)).
				Return(label, dtclient.ServerError{Code: 400, Message: "mock error"}).
				Once()

			return mockClient, nil
		}
		id, err := r.addToDashboard(tokenSecret, instance)
		assert.Error(t, err)
		assert.NotEmpty(t, id)
		assert.Equal(t, label, id)
	})
	t.Run("AddToDashboard no bearer token", func(t *testing.T) {
		tokenSecret.Data = map[string][]byte{}
		err := r.client.Update(context.TODO(), tokenSecret)
		assert.NoError(t, err)

		id, err := r.addToDashboard(tokenSecret, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "secret has no bearer token")
	})
	t.Run("AddToDashboard error building dynatrace client", func(t *testing.T) {
		r.dtcBuildFunc = func(rtc client.Client, instance *dynatracev1alpha1.ActiveGate, secret *corev1.Secret) (dtclient.Client, error) {
			return nil, fmt.Errorf("some error")
		}

		id, err := r.addToDashboard(tokenSecret, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "some error")
	})
	t.Run("AddToDashboard no token secret", func(t *testing.T) {
		err := r.client.Delete(context.TODO(), tokenSecret)
		assert.NoError(t, err)

		id, err := r.addToDashboard(tokenSecret, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "secrets \"dynatrace-activegate-secret\" not found")
	})
	t.Run("AddToDashboard malformed service account secret", func(t *testing.T) {
		serviceAccount.Secrets[0].Name = "wrong name"
		err := r.client.Update(context.TODO(), serviceAccount)
		assert.NoError(t, err)

		id, err := r.addToDashboard(tokenSecret, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "secrets \"wrong name\" not found")

		serviceAccount.Secrets[0].Name = ""
		err = r.client.Update(context.TODO(), serviceAccount)
		assert.NoError(t, err)

		id, err = r.addToDashboard(tokenSecret, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "bearer token name is empty")

		serviceAccount.Secrets = []corev1.ObjectReference{}
		err = r.client.Update(context.TODO(), serviceAccount)
		assert.NoError(t, err)

		id, err = r.addToDashboard(tokenSecret, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "could not find token name in service account secrets")
	})
	t.Run("AddToDashboard no service account", func(t *testing.T) {
		err := r.client.Delete(context.TODO(), serviceAccount)
		assert.NoError(t, err)

		id, err := r.addToDashboard(tokenSecret, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "serviceaccounts \"dynatrace-activegate\" not found")
	})
}

type addToDashboardLogger struct {
	logr.Logger
}

func (log addToDashboardLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	println("Called")
}
func (log addToDashboardLogger) Info(msg string, keysAndValues ...interface{}) {
	println("Called")
}

func TestHandleAddToDashboardResult(t *testing.T) {
	logger := addToDashboardLogger{}
	r, _, err := setupReconciler(t, &mockIsLatestUpdateService{})
	assert.NoError(t, err)

	r.handleAddToDashboardResult("some-id", nil, logger)
}
