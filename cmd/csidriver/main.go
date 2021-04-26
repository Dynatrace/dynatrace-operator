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
	"errors"
	"flag"
	"os"

	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/version"
	"github.com/spf13/pflag"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	nodeID   = flag.String("node-id", "", "node id")
	endpoint = flag.String("endpoint", "unix:///tmp/csi.sock", "CSI endpoint")

	log = logger.NewDTLogger().WithName("server")
)

var subcmdCallbacks = map[string]func(ns string, cfg *rest.Config) (manager.Manager, error){
	"csi-driver":            startCSIDriver,
	"csi-garbage-collector": startCSIGarbageCollector,
}

var errBadSubcmd = errors.New("subcommand must be csi-driver or csi-garbage-collector")

func main() {
	flag.Parse()
	ctrl.SetLogger(log)

	version.LogVersion()

	subcmd := "csi-driver"
	if args := pflag.Args(); len(args) > 0 {
		subcmd = args[0]
	}

	subcmdFn := subcmdCallbacks[subcmd]
	if subcmdFn == nil {
		log.Error(errBadSubcmd, "Unknown subcommand", "command", subcmd)
		os.Exit(1)
	}

	namespace := os.Getenv("POD_NAMESPACE")

	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	mgr, err := subcmdFn(namespace, cfg)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
