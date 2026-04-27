package operator

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core/middleware"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/system"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	use = "operator"
)

func New() *cobra.Command {
	return &cobra.Command{
		Use:          use,
		RunE:         run,
		SilenceUsage: true,
	}
}

func run(cmd *cobra.Command, args []string) error {
	installconfig.ReadModules()
	version.LogVersion()
	logd.LogBaseLoggerSettings()

	kubeCfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	if system.IsRunLocally() {
		log.Info("running locally in debug mode")

		return runLocally(ctrl.SetupSignalHandler(), kubeCfg)
	}

	return runInPod(kubeCfg)
}

func runInPod(kubeCfg *rest.Config) error {
	namespace := k8senv.DefaultNamespace()
	isOLM := system.IsDeployedViaOLM()

	operatorManager, err := createOperatorManager(kubeCfg, namespace, isOLM)
	if err != nil {
		return err
	}

	signalHandler := ctrl.SetupSignalHandler()
	go middleware.RunPeriodicCacheCleanup(signalHandler, k8senv.GetDTClientCacheCleanInterval(log))

	return errors.WithStack(operatorManager.Start(signalHandler))
}

func runLocally(ctx context.Context, kubeCfg *rest.Config) error {
	namespace := os.Getenv(k8senv.PodNamespace)

	if !system.IsDeployedViaOLM() {
		clt, err := client.New(kubeCfg, client.Options{Scheme: scheme.Scheme})
		if err != nil {
			return err
		}

		if err := certificates.InitReconcile(ctx, clt, namespace); err != nil {
			return err
		}
	}

	operatorManager, err := createOperatorManager(kubeCfg, namespace, false)
	if err != nil {
		return err
	}

	return errors.WithStack(operatorManager.Start(ctx))
}
