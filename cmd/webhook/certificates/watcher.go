package certificates

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/certificates"
	certsutils "github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// TODO: refactor code below to be testable and also tested.
const (
	certificateRenewalInterval = 6 * time.Hour
	// The folders will be readable and executed by others, but writable by the user only.
	permDirUser = 0775
	// Grants read and write permission to everyone.
	permAll     = 0666
	fiveMinutes = 5 * time.Minute
)

type CertificateWatcher struct {
	apiReader             client.Reader
	certificateDirectory  string
	namespace             string
	certificateSecretName string
}

func NewCertificateWatcher(mgr manager.Manager, namespace string, secretName string) (*CertificateWatcher, error) {
	webhookServer, ok := mgr.GetWebhookServer().(*webhook.DefaultServer)
	if !ok {
		return nil, errors.WithStack(errors.New("could not cast webhook server"))
	}

	return &CertificateWatcher{
		apiReader:             mgr.GetAPIReader(),
		certificateDirectory:  webhookServer.Options.CertDir,
		namespace:             namespace,
		certificateSecretName: secretName,
	}, nil
}

func (watcher *CertificateWatcher) watchForCertificatesSecret() {
	for {
		<-time.After(certificateRenewalInterval)
		log.Info("checking for new certificates")

		if updated, err := watcher.updateCertificatesFromSecret(); err != nil {
			log.Info("failed to update certificates", "error", err)
		} else if updated {
			log.Info("updated certificate successfully")
		}
	}
}

func (watcher *CertificateWatcher) updateCertificatesFromSecret() (bool, error) {
	var secret corev1.Secret

	err := watcher.apiReader.Get(context.TODO(),
		client.ObjectKey{Name: watcher.certificateSecretName, Namespace: watcher.namespace}, &secret)
	if err != nil {
		return false, err
	}

	if _, err = os.Stat(watcher.certificateDirectory); os.IsNotExist(err) {
		err = os.MkdirAll(watcher.certificateDirectory, permDirUser)
		if err != nil {
			return false, errors.WithMessage(err, "could not create cert directory")
		}
	}

	for _, filename := range []string{certificates.ServerCert, certificates.ServerKey} {
		if _, err = watcher.ensureCertificateFile(secret, filename); err != nil {
			return false, err
		}
	}

	isValid, err := certsutils.ValidateCertificateExpiration(secret.Data[certificates.ServerCert], certificateRenewalInterval, time.Now(), log)
	if err != nil {
		return false, err
	} else if !isValid {
		return false, errors.New("certificate is outdated")
	}

	return true, nil
}

func (watcher *CertificateWatcher) ensureCertificateFile(secret corev1.Secret, filename string) (bool, error) {
	f := filepath.Join(watcher.certificateDirectory, filename)

	data, err := os.ReadFile(f)
	if os.IsNotExist(err) || !bytes.Equal(data, secret.Data[filename]) {
		if err := os.WriteFile(f, secret.Data[filename], permAll); err != nil {
			return false, err
		}
	} else {
		return false, err
	}

	return true, nil
}

func (watcher *CertificateWatcher) WaitForCertificates() {
	for threshold := time.Now().Add(fiveMinutes); time.Now().Before(threshold); {
		_, err := watcher.updateCertificatesFromSecret()
		if err != nil {
			if k8serrors.IsNotFound(err) {
				log.Info("waiting for certificate secret to be available.")
			} else {
				log.Info("failed to update certificates", "error", err)
			}

			time.Sleep(10 * time.Second)

			continue
		}

		break
	}

	go watcher.watchForCertificatesSecret()
}
