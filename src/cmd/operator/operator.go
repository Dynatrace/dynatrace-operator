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
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/certificates"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/nodes"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/grzybek"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func setupBootstrapper(ns string, cfg *rest.Config, cancelMgr context.CancelFunc) (manager.Manager, error) {
	log.Info("starting certificate bootstrapper", "namespace", ns)
	mgr, err := setupBootstrapMgr(ns, cfg)
	if err != nil {
		return mgr, err
	}

	return mgr, certificates.AddBootstrap(mgr, ns, cancelMgr)
}

func setupOperator(ns string, cfg *rest.Config) (manager.Manager, error) {
	log.Info("starting operator", "namespace", ns)
	mgr, err := setupMgr(ns, cfg)
	if err != nil {
		return mgr, err
	}

	funcs := []func(manager.Manager, string) error{
		dynakube.Add,
		nodes.Add,
	}
	if !kubesystem.DeployedViaOLM() {
		funcs = append(funcs, certificates.Add)
	}

	for _, f := range funcs {
		if err := f(mgr, ns); err != nil {
			return nil, err
		}
	}

	return mgr, nil
}

func setupBootstrapMgr(ns string, cfg *rest.Config) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace: ns,
		Scheme:    scheme.Scheme,
	})
	if err != nil {
		return nil, err
	}

	return mgr, err
}

func setupMgr(ns string, cfg *rest.Config) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace:                  ns,
		Scheme:                     scheme.Scheme,
		MetricsBindAddress:         ":8080",
		Port:                       8383,
		LeaderElection:             true,
		LeaderElectionID:           "dynatrace-operator-lock",
		LeaderElectionResourceLock: "configmaps",
		LeaderElectionNamespace:    ns,
		HealthProbeBindAddress:     ":10080",
		LivenessEndpointName:       "/livez",
	})
	if err != nil {
		return nil, err
	}

	log.Info("registering manager components")
	if err = mgr.AddHealthzCheck("livez", grzybek.NewHttpRequestHandler(log)); err != nil {
		log.Error(err, "could not start health endpoint for operator")
	}

	if err = mgr.AddReadyzCheck("readyz", grzybek.NewHttpRequestHandler(log)); err != nil {
		log.Error(err, "could not start ready endpoint for operator")
	}
	return mgr, err
}
