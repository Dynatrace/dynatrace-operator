package server

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/webhook"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	webhook2 "sigs.k8s.io/controller-runtime/pkg/webhook"
)

// todo remove log
func UpdateCertificate(mgr manager.Manager, ws *webhook2.Server, ns string) error {
	var secret corev1.Secret

	err := mgr.GetAPIReader().Get(context.TODO(), client.ObjectKey{Name: webhook.SecretCertsName, Namespace: ns}, &secret)
	if err != nil {
		return err
	}

	if _, err := os.Stat(ws.CertDir); os.IsNotExist(err) {
		err = os.MkdirAll(ws.CertDir, 0755)
		return fmt.Errorf("could not create cert directory: %s", err)
	}
	// todo(gakr): don't pull from secrets if filename is different
	for _, filename := range []string{"tls.crt", "tls.key"} {
		f := filepath.Join(ws.CertDir, filename)

		data, err := ioutil.ReadFile(f)

		if os.IsNotExist(err) || !bytes.Equal(data, secret.Data[filename]) {
			if err := ioutil.WriteFile(f, secret.Data[filename], 0666); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
