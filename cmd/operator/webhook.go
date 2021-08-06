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
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/Dynatrace/dynatrace-operator/webhook/server"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func startWebhookServer(ns string, cfg *rest.Config) (manager.Manager, error) {
	mgr, err := newManagerWithCertificates(ns, cfg)
	if err != nil {
		return nil, err
	}

	waitForCertificates(newCertificateWatcher(mgr, ns, webhook.SecretCertsName))

	if err := server.AddToManager(mgr, ns); err != nil {
		return nil, err
	}

	return mgr, nil
}
