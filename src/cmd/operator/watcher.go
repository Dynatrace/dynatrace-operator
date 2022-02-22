/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const certificateRenewalInterval = 6 * time.Hour

type certificateWatcher struct {
	apiReader             client.Reader
	fs                    afero.Fs
	certificateDirectory  string
	namespace             string
	certificateSecretName string
}

func newCertificateWatcher(mgr manager.Manager, namespace string, secretName string) *certificateWatcher {
	return &certificateWatcher{
		apiReader:             mgr.GetAPIReader(),
		fs:                    afero.NewOsFs(),
		certificateDirectory:  certsDir,
		namespace:             namespace,
		certificateSecretName: secretName,
	}
}

func (watcher *certificateWatcher) watchForCertificatesSecret() {
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
		if _, err = watcher.ensureCertificateFile(secret, filename); err != nil {
			return false, err
		}
	}
	isValid, err := kubeobjects.ValidateCertificateExpiration(secret.Data["tls.crt"], certificateRenewalInterval, time.Now(), log)
	if err != nil {
		return false, err
	} else if !isValid {
		return false, fmt.Errorf("certificate is outdated")
	}
	return true, nil
}

func (watcher *certificateWatcher) ensureCertificateFile(secret corev1.Secret, filename string) (bool, error) {
	f := filepath.Join(watcher.certificateDirectory, filename)

	data, err := afero.ReadFile(watcher.fs, f)
	if os.IsNotExist(err) || !bytes.Equal(data, secret.Data[filename]) {
		if err := afero.WriteFile(watcher.fs, f, secret.Data[filename], 0666); err != nil {
			return false, err
		}
	} else {
		return false, err
	}
	return true, nil
}
