package main

import (
	"os"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	csidriver "github.com/Dynatrace/dynatrace-operator/controllers/csi/driver"
	csiprovisioner "github.com/Dynatrace/dynatrace-operator/controllers/csi/provisioner"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"golang.org/x/sys/unix"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func startCSIDriver(ns string, _ *rest.Config) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Namespace:          ns,
		Scheme:             scheme.Scheme,
		MetricsBindAddress: ":8686",
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	defaultUmask := unix.Umask(0002)
	defer unix.Umask(defaultUmask)

	csiOpts := dtcsi.CSIOptions{
		NodeID:   *nodeID,
		Endpoint: *endpoint,
		DataDir:  dtcsi.DataPath,
	}

	if err := os.MkdirAll(dtcsi.DataPath, 0770); err != nil {
		log.Error(err, "unable to create data directory for CSI Driver")
		os.Exit(1)
	}

	if err := os.MkdirAll(dtcsi.GarbageCollectionPath, 0770); err != nil {
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

	return mgr, nil
}
