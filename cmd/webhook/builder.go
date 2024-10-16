package webhook

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/cmd/certificates"
	"github.com/Dynatrace/dynatrace-operator/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/cmd/manager"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	edgeconnectv1alpha2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	dynakubev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube" //nolint:staticcheck
	dynakubev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube" //nolint:staticcheck
	dynakubev1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dynakubevalidation "github.com/Dynatrace/dynatrace-operator/pkg/api/validation/dynakube"
	edgeconnectvalidation "github.com/Dynatrace/dynatrace-operator/pkg/api/validation/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	namespacemutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/namespace"
	podmutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
	configProvider  config.Provider
	managerProvider cmdManager.Provider
	namespace       string
	podName         string
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
		builder.managerProvider = NewProvider(certificateDirectory, certificateKeyFileName, certificateFileName)
	}

	return builder.managerProvider
}

func (builder CommandBuilder) SetNamespace(namespace string) CommandBuilder {
	builder.namespace = namespace

	return builder
}

func (builder CommandBuilder) SetPodName(podName string) CommandBuilder {
	builder.podName = podName

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

func startCertificateWatcher(webhookManager manager.Manager, namespace string, podName string) error {
	webhookPod, err := pod.Get(context.TODO(), webhookManager.GetAPIReader(), podName, namespace)
	if err != nil {
		return err
	}

	isDeployedViaOLM := kubesystem.IsDeployedViaOlm(*webhookPod)
	if !isDeployedViaOLM {
		watcher, err := certificates.NewCertificateWatcher(webhookManager, namespace, webhook.SecretCertsName)
		if err != nil {
			return err
		}

		watcher.WaitForCertificates()
	}

	return nil
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		version.LogVersion()
		logd.LogBaseLoggerSettings()

		kubeConfig, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		webhookManager, err := builder.GetManagerProvider().CreateManager(builder.namespace, kubeConfig)
		if err != nil {
			return err
		}

		signalHandler := ctrl.SetupSignalHandler()

		err = startCertificateWatcher(webhookManager, builder.namespace, builder.podName)
		if err != nil {
			return err
		}

		err = namespacemutator.AddWebhookToManager(webhookManager, builder.namespace)
		if err != nil {
			return err
		}

		err = podmutator.AddWebhookToManager(signalHandler, webhookManager, builder.namespace)
		if err != nil {
			return err
		}

		dkValidator := dynakubevalidation.New(webhookManager.GetAPIReader(), webhookManager.GetConfig())

		err = dynakubev1beta1.SetupWebhookWithManager(webhookManager, dkValidator)
		if err != nil {
			return err
		}

		err = dynakubev1beta2.SetupWebhookWithManager(webhookManager, dkValidator)
		if err != nil {
			return err
		}

		err = dynakubev1beta3.SetupWebhookWithManager(webhookManager, dkValidator)
		if err != nil {
			return err
		}

		ecValidator := edgeconnectvalidation.New(webhookManager.GetAPIReader(), webhookManager.GetConfig())

		err = edgeconnectv1alpha1.SetupWebhookWithManager(webhookManager, ecValidator)
		if err != nil {
			return err
		}

		err = edgeconnectv1alpha2.SetupWebhookWithManager(webhookManager, ecValidator)
		if err != nil {
			return err
		}

		err = webhookManager.Start(signalHandler)

		return errors.WithStack(err)
	}
}
