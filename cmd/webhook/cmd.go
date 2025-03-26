package webhook

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/cmd/webhook/certificates"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	edgeconnectv1alpha2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	dynakubev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube" //nolint:staticcheck
	dynakubev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube" //nolint:staticcheck
	dynakubev1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dynakubev1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	dynakubevalidation "github.com/Dynatrace/dynatrace-operator/pkg/api/validation/dynakube"
	edgeconnectvalidation "github.com/Dynatrace/dynatrace-operator/pkg/api/validation/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	namespacemutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/namespace"
	podmutator "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&certificateDirectory, FlagCertificateDirectory, "/tmp/webhook/certs", "Directory to look certificates for.")
	cmd.PersistentFlags().StringVar(&certificateFileName, FlagCertificateFileName, "tls.crt", "File name for the public certificate.")
	cmd.PersistentFlags().StringVar(&certificateKeyFileName, FlagCertificateKeyFileName, "tls.key", "File name for the private key.")
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:          use,
		RunE:         run(),
		SilenceUsage: true,
	}

	addFlags(cmd)

	return cmd
}

func run() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		installconfig.ReadModules()
		version.LogVersion()
		logd.LogBaseLoggerSettings()

		podName := os.Getenv(env.PodName)
		namespace := os.Getenv(env.PodNamespace)

		kubeConfig, err := config.GetConfig()
		if err != nil {
			return err
		}

		webhookManager, err := createManager(kubeConfig, namespace, certificateDirectory, certificateFileName, certificateKeyFileName)
		if err != nil {
			return err
		}

		signalHandler := ctrl.SetupSignalHandler()

		err = startCertificateWatcher(webhookManager, namespace, podName)
		if err != nil {
			return err
		}

		err = namespacemutator.AddWebhookToManager(webhookManager, namespace)
		if err != nil {
			return err
		}

		err = podmutator.AddWebhookToManager(signalHandler, webhookManager, namespace)
		if err != nil {
			return err
		}

		err = setupDynakubeValidation(webhookManager)
		if err != nil {
			return err
		}

		err = setupEdgeconnectValidation(webhookManager)
		if err != nil {
			return err
		}

		err = webhookManager.Start(signalHandler)

		return errors.WithStack(err)
	}
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

func setupDynakubeValidation(webhookManager manager.Manager) error {
	dkValidator := dynakubevalidation.New(webhookManager.GetAPIReader(), webhookManager.GetConfig())

	err := dynakubev1beta1.SetupWebhookWithManager(webhookManager, dkValidator)
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

	err = dynakubev1beta4.SetupWebhookWithManager(webhookManager, dkValidator)
	if err != nil {
		return err
	}

	return nil
}

func setupEdgeconnectValidation(webhookManager manager.Manager) error {
	ecValidator := edgeconnectvalidation.New(webhookManager.GetAPIReader(), webhookManager.GetConfig())

	err := edgeconnectv1alpha1.SetupWebhookWithManager(webhookManager, ecValidator)
	if err != nil {
		return err
	}

	err = edgeconnectv1alpha2.SetupWebhookWithManager(webhookManager, ecValidator)
	if err != nil {
		return err
	}

	return nil
}
