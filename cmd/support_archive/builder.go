package support_archive

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"

	"github.com/Dynatrace/dynatrace-operator/cmd/config"
	"github.com/Dynatrace/dynatrace-operator/cmd/remote_command"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	clientgocorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

const (
	use                            = "support-archive"
	namespaceFlagName              = "namespace"
	archiveToStdoutFlagName        = "stdout"
	delayFlagName                  = "delay"
	defaultSupportArchiveTargetDir = "/tmp/dynatrace-operator"
	defaultOperatorAppName         = "dynatrace-operator"
	loadsimFileSizeFlagName        = "loadsim-file-size"
	loadsimFilesFlagName           = "loadsim-files"
	collectManagedLogsFlagName     = "managed-logs"
	defaultSimFileSize             = 10
)

const (
	_ = 1 << (10 * iota) //nolint:mnd
	Kibi
	Mebi
)

var (
	namespaceFlagValue          string
	archiveToStdoutFlagValue    bool
	loadsimFilesFlagValue       int
	loadsimFileSizeFlagValue    int
	collectManagedLogsFlagValue bool
	delayFlagValue              int
)

type CommandBuilder struct {
	configProvider config.Provider
}

func NewCommandBuilder() CommandBuilder {
	return CommandBuilder{}
}

func (builder CommandBuilder) SetConfigProvider(provider config.Provider) CommandBuilder {
	builder.configProvider = provider

	return builder
}

func clusterOptions(opts *cluster.Options) {
	opts.Scheme = scheme.Scheme
}

func (builder CommandBuilder) Build() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		Long: "Pack logs and manifests useful for troubleshooting into single tarball",
		RunE: builder.buildRun(),
	}
	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&namespaceFlagValue, namespaceFlagName, env.DefaultNamespace(), "Specify a different Namespace.")
	cmd.PersistentFlags().BoolVar(&archiveToStdoutFlagValue, archiveToStdoutFlagName, false, "Write tarball to stdout.")
	cmd.PersistentFlags().IntVar(&loadsimFileSizeFlagValue, loadsimFileSizeFlagName, defaultSimFileSize, "Simulated log files, size in MiB (default 10)")
	cmd.PersistentFlags().IntVar(&loadsimFilesFlagValue, loadsimFilesFlagName, 0, "Number of simulated log files (default 0)")
	cmd.PersistentFlags().BoolVar(&collectManagedLogsFlagValue, collectManagedLogsFlagName, true, "Add logs from rolled out pods to the support archive.")
	cmd.PersistentFlags().IntVar(&delayFlagValue, delayFlagName, 0, "Delay start of support-archive collection. Useful for standalone execution with 'kubectl run'")
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		time.Sleep(time.Duration(delayFlagValue) * time.Second)

		logBuffer := bytes.Buffer{}
		log := newSupportArchiveLogger(getLogOutput(archiveToStdoutFlagValue, &logBuffer))
		version.LogVersionToLogger(log)

		archiveTargetFile, err := createZipArchiveTargetFile(archiveToStdoutFlagValue, defaultSupportArchiveTargetDir)
		if err != nil {
			return err
		}

		supportArchive := newZipArchive(archiveTargetFile)

		defer archiveTargetFile.Close()
		defer supportArchive.Close()

		err = builder.runCollectors(log, supportArchive)
		if err != nil {
			return err
		}

		printCopyCommand(log, archiveToStdoutFlagValue, archiveTargetFile.Name())

		// make sure to run this collector at the very end
		newSupportArchiveOutputCollector(log, supportArchive, &logBuffer).Do()

		return nil
	}
}

func getLogOutput(tarballToStdout bool, logBuffer *bytes.Buffer) io.Writer {
	if tarballToStdout {
		// avoid corrupting tarball
		return io.MultiWriter(os.Stderr, logBuffer)
	} else {
		return io.MultiWriter(os.Stdout, logBuffer)
	}
}

func getAppNameLabel(ctx context.Context, pods clientgocorev1.PodInterface) string {
	podName := os.Getenv(env.PodName)
	if podName != "" {
		options := metav1.GetOptions{}

		pod, err := pods.Get(ctx, podName, options)
		if err != nil {
			return defaultOperatorAppName
		}

		return pod.Labels[labels.AppNameLabel]
	}

	return defaultOperatorAppName
}

func (builder CommandBuilder) runCollectors(log logd.Logger, supportArchive archiver) error {
	ctx := context.Background()

	kubeConfig, err := builder.configProvider.GetConfig()
	if err != nil {
		return err
	}

	clientSet, apiReader, err := getK8sClients(kubeConfig)
	if err != nil {
		return err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		return err
	}

	pods := clientSet.CoreV1().Pods(namespaceFlagValue)
	appName := getAppNameLabel(ctx, pods)

	logInfof(log, "%s=%s", labels.AppNameLabel, appName)

	fileSize := loadsimFileSizeFlagValue * Mebi
	collectors := []collector{
		newOperatorVersionCollector(log, supportArchive),
		newLogCollector(ctx, log, supportArchive, pods, appName, collectManagedLogsFlagValue),
		newFsLogCollector(ctx, kubeConfig, &remote_command.DefaultExecutor{}, log, supportArchive, pods, appName, collectManagedLogsFlagValue),
		newK8sObjectCollector(ctx, log, supportArchive, namespaceFlagValue, appName, apiReader, discoveryClient),
		newTroubleshootCollector(ctx, log, supportArchive, namespaceFlagValue, apiReader, *kubeConfig),
		newLoadSimCollector(ctx, log, supportArchive, fileSize, loadsimFilesFlagValue, clientSet.CoreV1().Pods(namespaceFlagValue)),
	}

	for _, c := range collectors {
		if err := c.Do(); err != nil {
			logErrorf(log, err, "%s failed", c.Name())
		}
	}

	return nil
}

func getK8sClients(kubeConfig *rest.Config) (*kubernetes.Clientset, client.Reader, error) {
	k8sCluster, err := cluster.New(kubeConfig, clusterOptions)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	clientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	apiReader := k8sCluster.GetAPIReader()

	return clientSet, apiReader, nil
}

func printCopyCommand(log logd.Logger, tarballToStdout bool, tarFileName string) {
	podNamespace := os.Getenv(env.PodNamespace)
	podName := os.Getenv(env.PodName)

	if tarballToStdout {
		return
	}

	if podNamespace == "" || podName == "" {
		// most probably not running on a pod
		logInfof(log, "cp %s .", tarFileName)
	} else {
		logInfof(log, "kubectl -n %s cp %s:%s .%s\n",
			podNamespace, podName, tarFileName, tarFileName)
	}
}
