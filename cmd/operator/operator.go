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

	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/controllers/namespace"
	"github.com/Dynatrace/dynatrace-operator/controllers/namespace_init"
	"github.com/Dynatrace/dynatrace-operator/controllers/nodes"
	"github.com/Dynatrace/dynatrace-operator/controllers/webhookcerts"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func startOperator(ns string, cfg *rest.Config) (manager.Manager, error) {
	log.Info(ns)
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace:                  ns,
		Scheme:                     scheme.Scheme,
		MetricsBindAddress:         ":8080",
		Port:                       8383,
		LeaderElection:             true,
		LeaderElectionID:           "dynatrace-operator-lock",
		LeaderElectionResourceLock: "configmaps",
		LeaderElectionNamespace:    ns,
		HealthProbeBindAddress:     "0.0.0.0:10080",
	})
	if err != nil {
		return nil, err
	}

	log.Info("Registering Components.")

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "could not start health endpoint for operator")
	}

	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "could not start ready endpoint for operator")
	}

	funcs := []func(manager.Manager, string) error{
		dynakube.Add,
		namespace.Add,
		nodes.Add,
		namespace_init.Add,
	}

	disableWebhook := os.Getenv("DISABLE_WEBHOOK")
	if disableWebhook == "" || disableWebhook == "false" {
		funcs = append(funcs, webhookcerts.Add)
	}

	for _, f := range funcs {
		if err := f(mgr, ns); err != nil {
			return nil, err
		}
	}

	return mgr, nil
}
