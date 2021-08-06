package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type certificateWatcher struct {
	apiReader             client.Reader
	fs                    afero.Fs
	certificateDirectory  string
	namespace             string
	certificateSecretName string
	logger                logr.Logger
}

func newCertificateWatcher(mgr manager.Manager, namespace string, secretName string) *certificateWatcher {
	return &certificateWatcher{
		apiReader:             mgr.GetAPIReader(),
		fs:                    afero.NewOsFs(),
		certificateDirectory:  certsDir,
		namespace:             namespace,
		certificateSecretName: secretName,
		logger:                log,
	}
}

func (watcher *certificateWatcher) watchForCertificatesSecret() {
	for {
		<-time.After(6 * time.Hour)
		watcher.logger.Info("checking for new certificates")
		if updated, err := watcher.updateCertificatesFromSecret(); err != nil {
			watcher.logger.Info("failed to update certificates", "error", err)
		} else if updated {
			watcher.logger.Info("updated certificate successfully")
		}
	}
}

func (watcher *certificateWatcher) updateCertificatesFromSecret() (bool, error) {
	var secret corev1.Secret

	err := watcher.apiReader.Get(context.TODO(),
		client.ObjectKey{Name: watcher.certificateSecretName, Namespace: watcher.namespace}, &secret)
	if err != nil {
		return false, err
	}

	if _, err := watcher.fs.Stat(watcher.certificateDirectory); os.IsNotExist(err) {
		err = watcher.fs.MkdirAll(watcher.certificateDirectory, 0755)
		if err != nil {
			return false, fmt.Errorf("could not create cert directory: %s", err)
		}
	}

	for _, filename := range []string{"tls.crt", "tls.key"} {
		f := filepath.Join(watcher.certificateDirectory, filename)

		data, err := afero.ReadFile(watcher.fs, f)

		if os.IsNotExist(err) || !bytes.Equal(data, secret.Data[filename]) {
			if err := afero.WriteFile(watcher.fs, f, secret.Data[filename], 0666); err != nil {
				return false, err
			}
		} else {
			return false, err
		}
	}

	return true, nil
}
