/*
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
	log = logger.NewDTLogger()
)

type subCommand func(string, *rest.Config) (mgr manager.Manager, cleanUp func(), err error)

var subcmdCallbacks = map[string]subCommand{
	"csi-driver":     startCSIDriver,
	"operator":       startOperator,
	"webhook-server": startWebhookServer,
}

var errBadSubcmd = errors.New("subcommand must be operator, or webhook-server")

func main() {

	pflag.CommandLine.AddFlagSet(webhookServerFlags())
	pflag.CommandLine.AddFlagSet(csiDriverFlags())
	pflag.Parse()

	ctrl.SetLogger(logger.NewDTLogger())

	version.LogVersion()

	subcmd := "operator"
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

	mgr, cleanUp, err := subcmdFn(namespace, cfg)
	defer cleanUp()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	signalHandler := ctrl.SetupSignalHandler()

	startWebhookAndBootstrapperIfDebugFlagSet(startupInfo{
		cfg:           cfg,
		namespace:     namespace,
		signalHandler: signalHandler,
	})

	log.Info("starting manager")
	if err := mgr.Start(signalHandler); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
