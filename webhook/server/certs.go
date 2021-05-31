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

func UpdateCertificate(mgr manager.Manager, ws *webhook2.Server, ns string) error {
	var secret corev1.Secret

	err := mgr.GetAPIReader().Get(context.TODO(), client.ObjectKey{Name: webhook.SecretCertsName, Namespace: ns}, &secret)
	if err != nil {
		return err
	}

	if _, err := os.Stat(ws.CertDir); os.IsNotExist(err) {
		err = os.MkdirAll(ws.CertDir, 0755)
		if err != nil {
			return fmt.Errorf("could not create cert directory: %s", err)
		}
	}

	for _, filename := range []string{"tls.crt", "tls.key"} {
		tmp_file := filepath.Join(ws.CertDir, filename+".tmp")
		orig_file := filepath.Join(ws.CertDir, filename)

		data, err := ioutil.ReadFile(orig_file)

		if os.IsNotExist(err) || !bytes.Equal(data, secret.Data[filename]) {
			// write to tmp and move file, otherwise certwatcher.go will read file before cert data is written
			if err := ioutil.WriteFile(tmp_file, secret.Data[filename], 0666); err != nil {
				return err
			}
			if err := os.Remove(orig_file); !os.IsNotExist(err) && err != nil {
				return err
			}
			if err := os.Rename(tmp_file, orig_file); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
