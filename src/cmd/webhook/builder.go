package webhook

import (
	"github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/certificates"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/namespace_mutator"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod_mutator"
	validationhook "github.com/Dynatrace/dynatrace-operator/src/webhook/validation"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

const use = "webhook-server"

type commandBuilder struct {
	configProvider   config.Provider
	managerProvider  cmdManager.Provider
	namespace        string
	isDeployedViaOlm bool
}

func newWebhookCommandBuilder() commandBuilder {
	return commandBuilder{}
}

func (builder commandBuilder) setConfigProvider(provider config.Provider) commandBuilder {
	builder.configProvider = provider
	return builder
}

func (builder commandBuilder) setManagerProvider(provider cmdManager.Provider) commandBuilder {
	builder.managerProvider = provider
	return builder
}

func (builder commandBuilder) setNamespace(namespace string) commandBuilder {
	builder.namespace = namespace
	return builder
}

func (builder commandBuilder) setIsDeployedViaOlm(isDeployedViaOlm bool) commandBuilder {
	builder.isDeployedViaOlm = isDeployedViaOlm
	return builder
}

func (builder commandBuilder) build() *cobra.Command {
	return &cobra.Command{
		Use:  use,
		RunE: builder.buildRun(),
	}
}

func (builder commandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// TODO: make the code below testable and test it, but in another ticket because otherwise adding the other commands will take a week
		kubeConfig, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		webhookManager, err := builder.managerProvider.CreateManager(builder.namespace, kubeConfig)
		if err != nil {
			return err
		}

		if !builder.isDeployedViaOlm {
			certificates.
				NewCertificateWatcher(webhookManager, builder.namespace, webhook.SecretCertsName).
				WaitForCertificates()
		}

		err = namespace_mutator.AddNamespaceMutationWebhookToManager(webhookManager, builder.namespace)
		if err != nil {
			return err
		}

		err = pod_mutator.AddPodMutationWebhookToManager(webhookManager, builder.namespace)
		if err != nil {
			return err
		}

		err = (&v1alpha1.DynaKube{}).SetupWebhookWithManager(webhookManager)
		if err != nil {
			return err
		}

		err = (&dynatracev1beta1.DynaKube{}).SetupWebhookWithManager(webhookManager)
		if err != nil {
			return err
		}

		err = validationhook.AddDynakubeValidationWebhookToManager(webhookManager)
		if err != nil {
			return err
		}

		signalHandler := ctrl.SetupSignalHandler()
		err = webhookManager.Start(signalHandler)

		return errors.WithStack(err)
	}
}
