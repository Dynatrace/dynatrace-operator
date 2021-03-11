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

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/version"
	"github.com/spf13/pflag"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	scheme = pkgruntime.NewScheme()
	log    = logger.NewDTLogger()
)

var subcmdCallbacks = map[string]func(ns string, cfg *rest.Config) (manager.Manager, error){
	"operator":             startOperator,
	"webhook-bootstrapper": startWebhookBoostrapper,
	"webhook-server":       startWebhookServer,
}

var errBadSubcmd = errors.New("subcommand must be operator, webhook-bootstrapper, or webhook-server")

var (
	certsDir string
	certFile string
	keyFile  string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(dynatracev1alpha1.AddToScheme(scheme))
	utilruntime.Must(istiov1alpha3.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	webhookServerFlags := pflag.NewFlagSet("webhook-server", pflag.ExitOnError)
	webhookServerFlags.StringVar(&certsDir, "certs-dir", "/mnt/webhook-certs", "Directory to look certificates for.")
	webhookServerFlags.StringVar(&certFile, "cert", "tls.crt", "File name for the public certificate.")
	webhookServerFlags.StringVar(&keyFile, "cert-key", "tls.key", "File name for the private key.")

	pflag.CommandLine.AddFlagSet(webhookServerFlags)
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

	mgr, err := subcmdFn(namespace, cfg)
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
