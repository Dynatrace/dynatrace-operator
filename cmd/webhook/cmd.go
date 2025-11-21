package webhook

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/cmd/webhook/certificates"
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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	use                        = "webhook-server"
	FlagCertificateDirectory   = "certs-dir"
	FlagCertificateFileName    = "cert"
	FlagCertificateKeyFileName = "cert-key"

	openshiftSecurityGVR = "security.openshift.io/v1"
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
		RunE:         run,
		SilenceUsage: true,
	}

	addFlags(cmd)

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	installconfig.ReadModules()
	version.LogVersion()
	logd.LogBaseLoggerSettings()

	podName := os.Getenv(env.PodName)
	namespace := os.Getenv(env.PodNamespace)

	kubeConfig, err := config.GetConfig()
	if err != nil {
		return err
	}

	isOpenShift := false

	client, err := discovery.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		logd.Get().WithName("platform").Error(err, "failed to detect platform, due to discovery client issues")
	} else {
		_, err = client.ServerResourcesForGroupVersion(openshiftSecurityGVR)
		switch {
		case err == nil:
			logd.Get().WithName("platform").Info("detected platform", "platform", "openshift")
			isOpenShift = true
		case k8serrors.IsNotFound(err):
			logd.Get().WithName("platform").Info("detected platform", "platform", "kubernetes")
		default:
			logd.Get().WithName("platform").Error(err, "failed to detect platform, defaulting to kubernetes")
		}
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

	err = podmutator.AddWebhookToManager(signalHandler, webhookManager, namespace, isOpenShift)
	if err != nil {
		return err
	}

	err = dynakubevalidation.SetupWebhookWithManager(webhookManager)
	if err != nil {
		return err
	}

	err = edgeconnectvalidation.SetupWebhookWithManager(webhookManager)
	if err != nil {
		return err
	}

	err = webhookManager.Start(signalHandler)

	return errors.WithStack(err)
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
