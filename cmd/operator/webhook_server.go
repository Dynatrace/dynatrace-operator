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
	"os"
	"path"
	"time"

	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/webhook/server"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func startWebhookServer(ns string, cfg *rest.Config) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace:                  ns,
		Scheme:                     scheme.Scheme,
		MetricsBindAddress:         ":8383",
		Port:                       8443,
		LeaderElection:             true,
		LeaderElectionID:           "dynatrace-webhook-server-lock",
		LeaderElectionResourceLock: "configmaps",
		LeaderElectionNamespace:    ns,
	})
	if err != nil {
		return nil, err
	}

	ws := mgr.GetWebhookServer()
	ws.CertDir = certsDir
	ws.KeyName = keyFile
	ws.CertName = certFile
	log.Info("SSL certificates configured", "dir", certsDir, "key", keyFile, "cert", certFile)

	// Wait until the certificates are available, otherwise the Manager will fail to start.
	certFilePath := path.Join(certsDir, certFile)
	for threshold := time.Now().Add(5 * time.Minute); time.Now().Before(threshold); {
		if _, err := os.Stat(certFilePath); os.IsNotExist(err) {
			log.Info("Waiting for certificates to be available.")
			time.Sleep(10 * time.Second)
			continue
		} else if err != nil {
			return nil, err
		}

		break
	}

	if err := server.AddToManager(mgr, ns); err != nil {
		return nil, err
	}

	return mgr, nil
}
