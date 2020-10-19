package activegate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dao"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("AddToDashboard", label, server.URL, string(tokenValue)).
			Return(label, nil)

		id, err := r.addToDashboard(dtc, instance)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Equal(t, label, id)
	})
	t.Run("AddToDashboard error from api", func(t *testing.T) {
		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("AddToDashboard", label, server.URL, string(tokenValue)).
			Return(label, dtclient.ServerError{Code: 400, Message: "mock error"}).
			Once()
		id, err := r.addToDashboard(dtc, instance)
		assert.Error(t, err)
		assert.NotEmpty(t, id)
		assert.Equal(t, label, id)
	})
	t.Run("AddToDashboard no bearer token", func(t *testing.T) {
		tokenSecret.Data = map[string][]byte{}
		err := r.client.Update(context.TODO(), tokenSecret)
		assert.NoError(t, err)

		dtc, err := createFakeDTClient(nil, nil, nil)
		assert.NoError(t, err)

		id, err := r.addToDashboard(dtc, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "secret has no bearer token")
	})
	t.Run("AddToDashboard no token secret", func(t *testing.T) {
		err := r.client.Delete(context.TODO(), tokenSecret)
		assert.NoError(t, err)

		dtc, err := createFakeDTClient(nil, nil, nil)
		assert.NoError(t, err)

		id, err := r.addToDashboard(dtc, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "secrets \"dynatrace-activegate-secret\" not found")
	})
	t.Run("AddToDashboard malformed service account secret", func(t *testing.T) {
		serviceAccount.Secrets[0].Name = "wrong name"
		err := r.client.Update(context.TODO(), serviceAccount)
		assert.NoError(t, err)

		dtc, err := createFakeDTClient(nil, nil, nil)
		assert.NoError(t, err)

		id, err := r.addToDashboard(dtc, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "secrets \"wrong name\" not found")

		serviceAccount.Secrets[0].Name = ""
		err = r.client.Update(context.TODO(), serviceAccount)
		assert.NoError(t, err)

		id, err = r.addToDashboard(dtc, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "bearer token name is empty")

		serviceAccount.Secrets = []corev1.ObjectReference{}
		err = r.client.Update(context.TODO(), serviceAccount)
		assert.NoError(t, err)

		id, err = r.addToDashboard(dtc, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "could not find token name in service account secrets")
	})
	t.Run("AddToDashboard no service account", func(t *testing.T) {
		err := r.client.Delete(context.TODO(), serviceAccount)
		assert.NoError(t, err)

		dtc, err := createFakeDTClient(nil, nil, nil)
		assert.NoError(t, err)

		id, err := r.addToDashboard(dtc, instance)
		assert.Empty(t, id)
		assert.EqualError(t, err, "serviceaccounts \"dynatrace-activegate\" not found")
	})
}

type addToDashboardLogger struct {
	logr.Logger
	loggedMessages      []string
	loggedErrors        []error
	loggedKeysAndValues []interface{}
}

func (log *addToDashboardLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	log.loggedErrors = append(log.loggedErrors, err)
	log.loggedMessages = append(log.loggedMessages, msg)
	log.loggedKeysAndValues = append(log.loggedKeysAndValues, keysAndValues...)
}
func (log *addToDashboardLogger) Info(msg string, keysAndValues ...interface{}) {
	log.loggedMessages = append(log.loggedMessages, msg)
	log.loggedKeysAndValues = append(log.loggedKeysAndValues, keysAndValues...)
}

func TestHandleAddToDashboardResult(t *testing.T) {
	r, _, err := setupReconciler(t, &mockIsLatestUpdateService{})
	assert.NoError(t, err)

	t.Run("HandleAddToDashboard", func(t *testing.T) {
		logger := &addToDashboardLogger{}

		r.handleAddToDashboardResult("some-id", nil, logger)
		assert.Equal(t, 1, len(logger.loggedMessages))
		assert.Equal(t, 2, len(logger.loggedKeysAndValues))
		assert.Empty(t, logger.loggedErrors)
		assert.Equal(t, "id", logger.loggedKeysAndValues[0])
		assert.Equal(t, "some-id", logger.loggedKeysAndValues[1])
		assert.Equal(t, "added ActiveGate to Kubernetes dashboard", logger.loggedMessages[0])
	})

	t.Run("HandleAddToDashboard no id", func(t *testing.T) {
		logger := &addToDashboardLogger{}

		r.handleAddToDashboardResult("", nil, logger)
		assert.Equal(t, 1, len(logger.loggedMessages))
		assert.Equal(t, 2, len(logger.loggedKeysAndValues))
		assert.Empty(t, logger.loggedErrors)
		assert.Equal(t, "id", logger.loggedKeysAndValues[0])
		assert.Equal(t, "<unset>", logger.loggedKeysAndValues[1])
		assert.Equal(t, "added ActiveGate to Kubernetes dashboard", logger.loggedMessages[0])
	})

	t.Run("HandleAddToDashboard any error", func(t *testing.T) {
		logger := &addToDashboardLogger{}

		r.handleAddToDashboardResult("some-id", fmt.Errorf("a random error"), logger)
		assert.Equal(t, 1, len(logger.loggedErrors))
		assert.Equal(t, 1, len(logger.loggedMessages))
		assert.Equal(t, 0, len(logger.loggedKeysAndValues))
		assert.Equal(t, "error when adding ActiveGate Kubernetes configuration", logger.loggedMessages[0])
		assert.EqualError(t, logger.loggedErrors[0], "a random error")
	})

	t.Run("HandleAddToDashboard any api error", func(t *testing.T) {
		logger := &addToDashboardLogger{}

		r.handleAddToDashboardResult("some-id", dtclient.ServerError{Code: 123, Message: "some error"}, logger)
		assert.Equal(t, 1, len(logger.loggedErrors))
		assert.Equal(t, 1, len(logger.loggedMessages))
		assert.Equal(t, 4, len(logger.loggedKeysAndValues))
		assert.Equal(t, "id", logger.loggedKeysAndValues[0])
		assert.Equal(t, "some-id", logger.loggedKeysAndValues[1])
		assert.Equal(t, "error", logger.loggedKeysAndValues[2])
		assert.Equal(t, "some error", logger.loggedKeysAndValues[3])
		assert.Equal(t, "error returned from Dynatrace API", logger.loggedMessages[0])
		assert.EqualError(t, logger.loggedErrors[0], "error returned from Dynatrace API")
	})

	t.Run("HandleAddToDashboard bad request api error", func(t *testing.T) {
		logger := &addToDashboardLogger{}

		r.handleAddToDashboardResult("some-id", dtclient.ServerError{Code: 400, Message: "entry already exists"}, logger)

		assert.Equal(t, 1, len(logger.loggedMessages))
		assert.Equal(t, 4, len(logger.loggedKeysAndValues))
		assert.Empty(t, logger.loggedErrors)
		assert.Equal(t, "id", logger.loggedKeysAndValues[0])
		assert.Equal(t, "some-id", logger.loggedKeysAndValues[1])
		assert.Equal(t, "error", logger.loggedKeysAndValues[2])
		assert.Equal(t, "entry already exists", logger.loggedKeysAndValues[3])
		assert.Equal(t, "error returned from Dynatrace API when adding ActiveGate Kubernetes configuration, ignore if configuration already exist", logger.loggedMessages[0])
	})
}
