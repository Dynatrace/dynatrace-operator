/*
Copyright 2021 Dynatrace LLC.

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
	"flag"
	"os"
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	csidriver "github.com/Dynatrace/dynatrace-operator/controllers/csi/driver"
	csigc "github.com/Dynatrace/dynatrace-operator/controllers/csi/gc"
	csiprovisioner "github.com/Dynatrace/dynatrace-operator/controllers/csi/provisioner"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/version"
	"golang.org/x/sys/unix"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

var (
	nodeID    = flag.String("node-id", "", "node id")
	endpoint  = flag.String("endpoint", "unix:///tmp/csi.sock", "CSI endpoint")
	probeAddr = flag.String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	log = logger.NewDTLogger().WithName("server")
)

func main() {
	flag.Parse()
	ctrl.SetLogger(log)

	version.LogVersion()

	namespace := os.Getenv("POD_NAMESPACE")

	defaultUmask := unix.Umask(0002)
	defer unix.Umask(defaultUmask)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Namespace:              namespace,
		Scheme:                 scheme.Scheme,
		HealthProbeBindAddress: *probeAddr,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	csiOpts := dtcsi.CSIOptions{
		NodeID:   *nodeID,
		Endpoint: *endpoint,
		RootDir:  "/tmp",
	}

	if err := os.MkdirAll(filepath.Join(csiOpts.RootDir, dtcsi.DataPath), 0770); err != nil {
		log.Error(err, "unable to create data directory for CSI Driver")
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Join(csiOpts.RootDir, dtcsi.GarbageCollectionPath), 0770); err != nil {
		log.Error(err, "unable to create garbage collector directory for CSI Driver")
		os.Exit(1)
	}

	if err := csidriver.NewServer(mgr, csiOpts).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create CSI Driver server")
		os.Exit(1)
	}

	if err := csiprovisioner.NewReconciler(mgr, csiOpts).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create CSI Provisioner")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err := csigc.NewReconciler(mgr.GetClient(), csiOpts).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create CSI Garbage Collector")
		os.Exit(1)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
