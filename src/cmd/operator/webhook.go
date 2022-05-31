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
	"github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/namespace_mutator"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod_mutator"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	certsDir string
	certFile string
	keyFile  string
)

func webhookServerFlags() *pflag.FlagSet {
	webhookServerFlagSet := pflag.NewFlagSet("webhook-server", pflag.ExitOnError)
	webhookServerFlagSet.StringVar(&certsDir, "certs-dir", "/tmp/webhook/certs", "Directory to look certificates for.")
	webhookServerFlagSet.StringVar(&certFile, "cert", "tls.crt", "File name for the public certificate.")
	webhookServerFlagSet.StringVar(&keyFile, "cert-key", "tls.key", "File name for the private key.")
	return webhookServerFlagSet
}

func setupWebhookServer(ns string, cfg *rest.Config) (manager.Manager, func(), error) {
	mgr, cleanUp, err := newManagerWithCertificates(ns, cfg)
	if err != nil {
		return nil, cleanUp, err
	}

	if !kubesystem.DeployedViaOLM() {
		waitForCertificates(newCertificateWatcher(mgr, ns, webhook.SecretCertsName))
	}

	if err := namespace_mutator.AddNamespaceMutationWebhookToManager(mgr, ns); err != nil {
		return nil, cleanUp, err
	}

	if err := pod_mutator.AddPodMutationWebhookToManager(mgr, ns); err != nil {
		return nil, cleanUp, err
	}

	if err := (&v1alpha1.DynaKube{}).SetupWebhookWithManager(mgr); err != nil {
		return nil, cleanUp, err
	}

	if err := (&dynatracev1beta1.DynaKube{}).SetupWebhookWithManager(mgr); err != nil {
		return nil, cleanUp, err
	}

	return startValidationServer(mgr)
}
