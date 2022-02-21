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
	"context"
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/spf13/pflag"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	log = logger.NewDTLogger().WithName("main")
)

const (
	operatorCmd      = "operator"
	csiDriverCmd     = "csi-driver"
	standaloneCmd    = "init"
	webhookServerCmd = "webhook-server"
)

var errBadSubcmd = fmt.Errorf("subcommand must be %s, %s, %s or %s", operatorCmd, csiDriverCmd, webhookServerCmd, standaloneCmd)

func main() {
	pflag.CommandLine.AddFlagSet(webhookServerFlags())
	pflag.CommandLine.AddFlagSet(csiDriverFlags())
	pflag.Parse()

	ctrl.SetLogger(log)

	version.LogVersion()

	namespace := os.Getenv("POD_NAMESPACE")
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	var mgr manager.Manager
	var cleanUp func()

	subCmd := getSubCommand()
	switch subCmd {
	case operatorCmd:
		if !kubesystem.DeployedViaOLM() {
			// setup manager only for certificates
			bootstrapperCtx, done := context.WithCancel(context.TODO())
			mgr, err = setupBootstrapper(namespace, cfg, done)
			exitOnError(err, "bootstrapper setup failed")
			exitOnError(mgr.Start(bootstrapperCtx), "problem running bootstrap manager")
		}
		// bootstrap manager stopped, starting full manager
		mgr, err = setupOperator(namespace, cfg)
		exitOnError(err, "operator setup failed")
	case csiDriverCmd:
		mgr, cleanUp, err = setupCSIDriver(namespace, cfg)
		exitOnError(err, "csi driver setup failed")
		defer cleanUp()
	case webhookServerCmd:
		mgr, cleanUp, err = setupWebhookServer(namespace, cfg)
		exitOnError(err, "webhook-server setup failed")
		defer cleanUp()
	case standaloneCmd:
		err = startStandAloneInit()
		exitOnError(err, "initContainer command failed")
		os.Exit(0)
	default:
		log.Error(errBadSubcmd, "unknown subcommand", "command", subCmd)
		os.Exit(1)
	}

	signalHandler := ctrl.SetupSignalHandler()
	log.Info("starting manager")
	exitOnError(mgr.Start(signalHandler), "problem running manager")
}

func exitOnError(err error, msg string) {
	if err != nil {
		log.Error(err, msg)
		os.Exit(1)
	}
}

func getSubCommand() string {
	if args := pflag.Args(); len(args) > 0 {
		return args[0]
	}
	return operatorCmd
}
