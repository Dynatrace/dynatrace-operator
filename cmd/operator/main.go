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
	"errors"
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/version"
	"github.com/spf13/pflag"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	log = logger.NewDTLogger()
)

const (
	operatorCmd      = "operator"
	csiDriverCmd     = "csi-driver"
	webhookServerCmd = "webhook-server"
)

var errBadSubcmd = errors.New(fmt.Sprintf("subcommand must be %s, %s or %s", operatorCmd, csiDriverCmd, webhookServerCmd))

func main() {
	pflag.CommandLine.AddFlagSet(webhookServerFlags())
	pflag.CommandLine.AddFlagSet(csiDriverFlags())
	pflag.Parse()

	ctrl.SetLogger(logger.NewDTLogger())

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
		// start manager only for certificates
		bootstrapperCtx, done := context.WithCancel(context.TODO())
		mgr, err = startBootstrapper(namespace, cfg, done)
		exitOnError(err, "bootstrapper could not be configured")
		exitOnError(mgr.Start(bootstrapperCtx), "problem running bootstrap manager")
		// bootstrap manager stopped, starting full manager
		mgr, err = startOperator(namespace, cfg)
	case csiDriverCmd:
		mgr, cleanUp, err = startCSIDriver(namespace, cfg)
		exitOnError(err, "CSIDriver startup failed")
		defer cleanUp()
	case webhookServerCmd:
		mgr, cleanUp, err = startWebhookServer(namespace, cfg)
		exitOnError(err, "webhook-server startup failed")
		defer cleanUp()
	default:
		log.Error(errBadSubcmd, "Unknown subcommand", "command", subCmd)
		os.Exit(1)
	}

	signalHandler := ctrl.SetupSignalHandler()
	startWebhookIfDebugFlagSet(startupInfo{
		cfg:           cfg,
		namespace:     namespace,
		signalHandler: signalHandler,
	})

	log.Info("starting manager")
	exitOnError(mgr.Start(signalHandler), "problem running manager")
}

func exitOnError(err error, msg string, keysAndValues ...interface{}) {
	if err != nil {
		log.Error(err, msg, keysAndValues)
		os.Exit(1)
	}
}

func getSubCommand() string {
	if args := pflag.Args(); len(args) > 0 {
		return args[0]
	}
	return operatorCmd
}
