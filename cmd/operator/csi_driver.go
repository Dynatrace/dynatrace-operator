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
	"os"
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	csidriver "github.com/Dynatrace/dynatrace-operator/controllers/csi/driver"
	csigc "github.com/Dynatrace/dynatrace-operator/controllers/csi/gc"
	"github.com/Dynatrace/dynatrace-operator/controllers/csi/metadata"
	csiprovisioner "github.com/Dynatrace/dynatrace-operator/controllers/csi/provisioner"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"golang.org/x/sys/unix"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	nodeID    string
	endpoint  string
	probeAddr string
)

func csiDriverFlags() *pflag.FlagSet {
	csiDriverFlags := pflag.NewFlagSet("csi-driver", pflag.ExitOnError)
	csiDriverFlags.StringVar(&nodeID, "node-id", "", "node id")
	csiDriverFlags.StringVar(&endpoint, "endpoint", "unix:///tmp/csi.sock", "CSI endpoint")
	csiDriverFlags.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	return csiDriverFlags
}

func startCSIDriver(ns string, cfg *rest.Config) (manager.Manager, func(), error) {
	defaultUmask := unix.Umask(0000)
	cleanUp := func() {
		unix.Umask(defaultUmask)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace:              ns,
		Scheme:                 scheme.Scheme,
		MetricsBindAddress:     ":8080",
		Port:                   8383,
		HealthProbeBindAddress: probeAddr,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		return nil, cleanUp, err
	}

	csiOpts := dtcsi.CSIOptions{
		NodeID:   nodeID,
		Endpoint: endpoint,
		RootDir:  dtcsi.DataPath,
	}

	fs := afero.NewOsFs()

	if err := fs.MkdirAll(filepath.Join(csiOpts.RootDir), 0770); err != nil {
		log.Error(err, "unable to create data directory for CSI Driver")
		return nil, cleanUp, err
	}

	access, err := metadata.NewAccess(dtcsi.MetadataAccessPath)
	if err != nil {
		log.Error(err, "failed to setup database storage for CSI Driver")
		os.Exit(1)
	}
	if err := metadata.CorrectMetadata(mgr.GetClient(), access, log); err != nil {
		log.Error(err, "failed to correct database storage for CSI Driver")
	}

	if err := csidriver.NewServer(mgr.GetClient(), csiOpts, access).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create CSI Driver server")
		return nil, cleanUp, err
	}

	if err := csiprovisioner.NewReconciler(mgr, csiOpts, access).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create CSI Provisioner")
		return nil, cleanUp, err
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up health check")
		return nil, cleanUp, err
	}

	if err := csigc.NewReconciler(mgr.GetClient(), csiOpts, access).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create CSI Garbage Collector")
		return nil, cleanUp, err
	}

	return mgr, cleanUp, nil
}
