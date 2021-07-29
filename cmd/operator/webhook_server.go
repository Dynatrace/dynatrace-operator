/*
Copyright 2020 Dynatrace LLC.

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
	"time"

	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/webhook/server"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	certsDir string
	certFile string
	keyFile  string
)

func webhookServerFlags() *pflag.FlagSet {
	webhookServerFlags := pflag.NewFlagSet("webhook-server", pflag.ExitOnError)
	webhookServerFlags.StringVar(&certsDir, "certs-dir", "/tmp/webhook/certs", "Directory to look certificates for.")
	webhookServerFlags.StringVar(&certFile, "cert", "tls.crt", "File name for the public certificate.")
	webhookServerFlags.StringVar(&keyFile, "cert-key", "tls.key", "File name for the private key.")
	return webhookServerFlags
}

func startWebhookServer(ns string, cfg *rest.Config) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace:          ns,
		Scheme:             scheme.Scheme,
		MetricsBindAddress: ":8383",
		Port:               8443,
	})
	if err != nil {
		return nil, err
	}

	ws := mgr.GetWebhookServer()
	ws.CertDir = certsDir
	ws.KeyName = keyFile
	ws.CertName = certFile
	log.Info("SSL certificates configured", "dir", certsDir, "key", keyFile, "cert", certFile)
	fs := afero.NewOsFs()

	for threshold := time.Now().Add(5 * time.Minute); time.Now().Before(threshold); {
		_, err := server.UpdateCertificate(mgr.GetAPIReader(), fs, ws.CertDir, ns)

		if err != nil {
			if k8serrors.IsNotFound(err) {
				log.Info("Waiting for certificate secret to be available.")
			} else {
				log.Info("Failed to update certificates", "error", err)
			}
			time.Sleep(10 * time.Second)
			continue
		}

		break
	}
	startCertificateWatcher(mgr.GetAPIReader(), fs, ws.CertDir, ns)

	if err := server.AddToManager(mgr, ns); err != nil {
		return nil, err
	}

	return mgr, nil
}

func startCertificateWatcher(apiReader client.Reader, fs afero.Fs, certDir string, ns string) {
	go func() {
		for {
			<-time.After(6 * time.Hour)
			log.Info("checking for new certificates")
			if updated, err := server.UpdateCertificate(apiReader, fs, certDir, ns); err != nil {
				log.Info("failed to update certificates", "error", err)
			} else if updated {
				log.Info("updated certificate successfully")
			}
		}
	}()

}
