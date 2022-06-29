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

const (
	use                        = "webhook-server"
	FlagCertificateDirectory   = "certs-dir"
	FlagCertificateFileName    = "cert"
	FlagCertificateKeyFileName = "cert-key"
)

var (
	certificateDirectory   string
	certificateFileName    string
	certificateKeyFileName string
)

type CommandBuilder struct {
	configProvider   config.Provider
	managerProvider  cmdManager.Provider
	namespace        string
	isDeployedViaOlm bool
}

func NewWebhookCommandBuilder() CommandBuilder {
	return CommandBuilder{}
}

func (builder CommandBuilder) SetConfigProvider(provider config.Provider) CommandBuilder {
	builder.configProvider = provider
	return builder
}

func (builder CommandBuilder) SetManagerProvider(provider cmdManager.Provider) CommandBuilder {
	builder.managerProvider = provider
	return builder
}

func (builder CommandBuilder) GetManagerProvider() cmdManager.Provider {
	if builder.managerProvider == nil {
		builder.managerProvider = NewWebhookManagerProvider(certificateDirectory, certificateKeyFileName, certificateFileName)
	}

	return builder.managerProvider
}

func (builder CommandBuilder) SetNamespace(namespace string) CommandBuilder {
	builder.namespace = namespace
	return builder
}

func (builder CommandBuilder) SetIsDeployedViaOlm(isDeployedViaOlm bool) CommandBuilder {
	builder.isDeployedViaOlm = isDeployedViaOlm
	return builder
}

func (builder CommandBuilder) Build() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		RunE: builder.buildRun(),
	}

	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&certificateDirectory, FlagCertificateDirectory, "/tmp/webhook/certs", "Directory to look certificates for.")
	cmd.PersistentFlags().StringVar(&certificateFileName, FlagCertificateFileName, "tls.crt", "File name for the public certificate.")
	cmd.PersistentFlags().StringVar(&certificateKeyFileName, FlagCertificateKeyFileName, "tls.key", "File name for the private key.")
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// TODO: make the code below testable and test it, but in another ticket because otherwise adding the other commands will take a week
		kubeConfig, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		webhookManager, err := builder.GetManagerProvider().CreateManager(builder.namespace, kubeConfig)
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
