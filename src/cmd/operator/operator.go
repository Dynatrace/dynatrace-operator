package operator

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/certificates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	use = "operator"
)

type runConfig struct {
	kubeConfigProvider       configProvider
	bootstrapManagerProvider managerProvider
	operatorManagerProvider  managerProvider
	isDeployedInOlm          bool
	namespace                string
	signalHandler            context.Context
}

func newOperatorCommand(runCfg runConfig) *cobra.Command {
	return &cobra.Command{
		Use:  use,
		RunE: run(runCfg),
	}
}

func run(runCfg runConfig) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		kubeCfg, err := runCfg.kubeConfigProvider.GetConfig()

		if err != nil {
			return err
		}

		if !runCfg.isDeployedInOlm {
			err = runBootstrapper(runCfg, kubeCfg)
			if err != nil {
				return err
			}
		}

		operatorManager, err := runCfg.operatorManagerProvider.CreateManager(runCfg.namespace, kubeCfg)

		if err != nil {
			return err
		}

		err = operatorManager.Start(runCfg.signalHandler)

		return errors.WithStack(err)
	}
}

func runBootstrapper(runCfg runConfig, kubeCfg *rest.Config) error {
	var bootstrapManager ctrl.Manager
	bootstrapManager, err := runCfg.bootstrapManagerProvider.CreateManager(runCfg.namespace, kubeCfg)

	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(context.TODO())
	err = certificates.AddBootstrap(bootstrapManager, runCfg.namespace, cancelFn)

	if err != nil {
		return errors.WithStack(err)
	}

	err = bootstrapManager.Start(ctx)

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
