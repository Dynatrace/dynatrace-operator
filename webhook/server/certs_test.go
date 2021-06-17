package server

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/controllers/webhookcerts"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	ns       = "test-ns"
	domain   = "dynatrace-oneagent-webhook.webhook.svc"
	certsDir = "/tmp/certs"
)

func TestUpdateWebhookCertificate(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout))

	// generate certs for testing and save in secret
	now, _ := time.Parse(time.RFC3339, "2018-01-10T00:00:00Z")
	secret, err := generateCerts(now, logger)
	assert.NoError(t, err)
	fakeClient := fake.NewClient(&secret)

	// check if certs are updated
	fs := afero.NewMemMapFs()

	updated, err := UpdateCertificate(fakeClient, fs, certsDir, ns)
	assert.NoError(t, err)
	assert.True(t, updated)

	// check if certs are downloaded
	for _, filename := range []string{"tls.crt", "tls.key"} {
		exists, err := afero.Exists(fs, path.Join(certsDir, filename))
		assert.NoError(t, err)
		assert.True(t, exists)
	}

	// check if called again update is false
	updated, err = UpdateCertificate(fakeClient, fs, certsDir, ns)
	assert.NoError(t, err)
	assert.False(t, updated)

	// check if new certificates get downloaded if available
	now, _ = time.Parse(time.RFC3339, "2020-01-10T00:00:00Z")
	secret, err = generateCerts(now, logger)
	assert.NoError(t, err)
	assert.NoError(t, fakeClient.Update(context.TODO(), &secret))

	updated, err = UpdateCertificate(fakeClient, fs, certsDir, ns)
	assert.NoError(t, err)
	assert.True(t, updated)

	// check if there is an error if secret is no longer there
	assert.NoError(t, fakeClient.Delete(context.TODO(), &secret))
	updated, err = UpdateCertificate(fakeClient, fs, certsDir, ns)
	assert.Error(t, err)
	assert.False(t, updated)

	// check if certs in directory were not deleted
	for _, filename := range []string{"tls.crt", "tls.key"} {
		exists, err := afero.Exists(fs, path.Join(certsDir, filename))
		assert.NoError(t, err)
		assert.True(t, exists)
	}
}

func generateCerts(now time.Time, logger logr.Logger) (corev1.Secret, error) {
	validCerts := webhookcerts.Certs{Log: logger, Domain: domain, Now: now}
	if err := validCerts.ValidateCerts(); err != nil {
		return corev1.Secret{}, err
	}

	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhook.SecretCertsName,
			Namespace: ns,
		},
		Data: validCerts.Data,
	}, nil
}
