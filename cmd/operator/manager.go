package main

import (
	"time"

	"github.com/Dynatrace/dynatrace-operator/scheme"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func newManagerWithCertificates(ns string, cfg *rest.Config) (manager.Manager, error) {
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
	return mgr, nil
}

func waitForCertificates(watcher *certificateWatcher) {
	for threshold := time.Now().Add(5 * time.Minute); time.Now().Before(threshold); {
		_, err := watcher.updateCertificatesFromSecret()

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
	go watcher.watchForCertificatesSecret()
}
