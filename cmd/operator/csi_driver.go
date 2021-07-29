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
	"strconv"
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	csidriver "github.com/Dynatrace/dynatrace-operator/controllers/csi/driver"
	csigc "github.com/Dynatrace/dynatrace-operator/controllers/csi/gc"
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

func startCSIDriver(ns string, cfg *rest.Config) (manager.Manager, error) {
	gcInterval, err := strconv.Atoi(os.Getenv("GC_INTERVAL_MINUTES"))
	if err != nil {
		log.Error(err, "unable to convert GC_INTERVAL_MINUTES to int")
		return nil, err
	}

	defaultUmask := unix.Umask(0000)
	defer unix.Umask(defaultUmask)

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace:              ns,
		Scheme:                 scheme.Scheme,
		MetricsBindAddress:     ":8080",
		Port:                   8383,
		HealthProbeBindAddress: probeAddr,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		return nil, err
	}

	csiOpts := dtcsi.CSIOptions{
		NodeID:     nodeID,
		Endpoint:   endpoint,
		RootDir:    dtcsi.DataPath,
		GCInterval: time.Duration(gcInterval) * time.Minute,
	}

	fs := afero.NewOsFs()

	if err := fs.MkdirAll(filepath.Join(csiOpts.RootDir), 0770); err != nil {
		log.Error(err, "unable to create data directory for CSI Driver")
		return nil, err
	}

	if err := fs.MkdirAll(filepath.Join(csiOpts.RootDir, dtcsi.GarbageCollectionPath), 0770); err != nil {
		log.Error(err, "unable to create garbage collector directory for CSI Driver")
		return nil, err
	}

	if err := csidriver.NewServer(mgr.GetClient(), csiOpts).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create CSI Driver server")
		return nil, err
	}

	if err := csiprovisioner.NewReconciler(mgr, csiOpts).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create CSI Provisioner")
		return nil, err
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up health check")
		return nil, err
	}

	if err := csigc.NewReconciler(mgr.GetClient(), csiOpts).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create CSI Garbage Collector")
		return nil, err
	}

	return mgr, nil
}
