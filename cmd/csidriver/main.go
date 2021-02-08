/*
Copyright 2017 The Kubernetes Authors.

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
	"fmt"
	"os"
	"runtime"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	csidriver "github.com/Dynatrace/dynatrace-operator/controllers/csi/driver"
	csiprovisioner "github.com/Dynatrace/dynatrace-operator/controllers/csi/provisioner"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"golang.org/x/sys/unix"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
)

const driverName = "csi.oneagent.dynatrace.com"
const driverVersion = "snapshot"

var (
	nodeID                   = flag.String("node-id", "", "node id")
	endpoint                 = flag.String("endpoint", "unix:///tmp/csi.sock", "CSI endpoint")
	allowedSupportNamespaces = flag.String("allowed-support-namespaces", "",
		"Comma-separated list of namespaces that are allowed to access support-format volumes")

	scheme = k8sruntime.NewScheme()
	log    = logger.NewDTLogger().WithName("server")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(dynatracev1alpha1.AddToScheme(scheme))
}

func main() {
	flag.Parse()

	ctrl.SetLogger(log)

	printVersion()

	ns := os.Getenv("POD_NAMESPACE")

	defaultUmask := unix.Umask(0002)
	defer unix.Umask(defaultUmask)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Namespace: ns,
		Scheme:    scheme,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	supportNamespaces := strings.Split(*allowedSupportNamespaces, ",")

	if err := csidriver.NewServer(mgr, *nodeID, *endpoint, "/tmp/data", supportNamespaces).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create server", "server", "csi-driver")
		os.Exit(1)
	}

	if err := csiprovisioner.NewReconciler(mgr, "/tmp/data").SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "DynaKube")
		os.Exit(1)
	}

	signalHandler := ctrl.SetupSignalHandler()

	log.Info("starting manager")
	if err := mgr.Start(signalHandler); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", driverVersion))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}
