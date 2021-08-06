package main

import (
	"github.com/Dynatrace/dynatrace-operator/controllers/certificates/validation"
	validationhook "github.com/Dynatrace/dynatrace-operator/webhook/validation"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func startValidationServer(ns string, cfg *rest.Config) (manager.Manager, error) {
	mgr, err := newManagerWithCertificates(ns, cfg)
	if err != nil {
		return mgr, err
	}

	waitForCertificates(newCertificateWatcher(mgr, ns, validation.SecretCertsName))

	if err = validationhook.AddDynakubeValidationWebhookToManager(mgr); err != nil {
		return nil, err
	}

	return mgr, nil
}
