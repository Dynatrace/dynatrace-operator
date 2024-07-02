package webhook

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/cmd/certificates"
	"github.com/Dynatrace/dynatrace-operator/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/cmd/manager"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dynatracev1beta2validation "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube/validation"
	dynatracev1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	namespacemutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/namespace"
	podmutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod"
	edgeconnectvalidationhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/validation/edgeconnect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

		// instrument webhook manager HTTP client with OpenTelemetry
		webhookManager.GetHTTPClient().Transport = otelhttp.NewTransport(webhookManager.GetHTTPClient().Transport)

		signalHandler := ctrl.SetupSignalHandler()

		otelShutdownFn := dtotel.Start(signalHandler, "dynatrace-webhook", webhookManager.GetAPIReader(), builder.namespace)
		defer otelShutdownFn()

		err = startCertificateWatcher(webhookManager, builder.namespace, builder.podName)
		if err != nil {
			return err
		}

		err = namespacemutator.AddWebhookToManager(webhookManager, builder.namespace)
		if err != nil {
			return err
		}

		err = podmutator.AddWebhookToManager(webhookManager, builder.namespace)
		if err != nil {
			return err
		}

		err = (&dynatracev1beta1.DynaKube{}).SetupWebhookWithManager(webhookManager)
		if err != nil {
			return err
		}

		dkv1beta2Validator := dynatracev1beta2validation.New(webhookManager.GetAPIReader(), webhookManager.GetConfig())

		err = dynatracev1beta2.SetupWebhookWithManager(webhookManager, dkv1beta2Validator)
		if err != nil {
			return err
		}

		err = (&dynatracev1beta3.DynaKube{}).SetupWebhookWithManager(webhookManager)
		if err != nil {
			return err
		}

		err = edgeconnectvalidationhook.AddEdgeConnectValidationWebhookToManager(webhookManager)
		if err != nil {
			return err
		}

		err = webhookManager.Start(signalHandler)

		return errors.WithStack(err)
	}
}
