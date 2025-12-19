package supportarchive

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/cmd/supportarchive/remotecommand"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	clientgocorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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
	numEventsFlagName              = "num-events"
	defaultSimFileSize             = 10
	DefaultNumEvents               = 300
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
	NumEventsFlagValue          int
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		Long: "Pack logs and manifests useful for troubleshooting into single tarball",
		RunE: run,
		Args: func(cmd *cobra.Command, args []string) error {
			if archiveToStdoutFlagValue {
				return nil
			}

			sb := strings.Builder{}
			sb.WriteString("The only option to retrieve the support archive is by using '--stdout=true'. ")
			sb.WriteString("Please provide this parameter and make sure that you pipe the command output to a file. ")
			sb.WriteString("Otherwise, your terminal will be flooded with binary data.")

			return errors.New(sb.String())
		},
	}
	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&namespaceFlagValue, namespaceFlagName, k8senv.DefaultNamespace(), "Specify a different Namespace.")
	cmd.PersistentFlags().BoolVar(&archiveToStdoutFlagValue, archiveToStdoutFlagName, false, "Write tarball to stdout.")
	cmd.PersistentFlags().IntVar(&loadsimFileSizeFlagValue, loadsimFileSizeFlagName, defaultSimFileSize, "Simulated log files, size in MiB (default 10)")
	cmd.PersistentFlags().IntVar(&loadsimFilesFlagValue, loadsimFilesFlagName, 0, "Number of simulated log files (default 0)")
	cmd.PersistentFlags().BoolVar(&collectManagedLogsFlagValue, collectManagedLogsFlagName, true, "Add logs from rolled out pods to the support archive.")
	cmd.PersistentFlags().IntVar(&delayFlagValue, delayFlagName, 0, "Delay start of support-archive collection. Useful for standalone execution with 'kubectl run'")
	cmd.PersistentFlags().IntVar(&NumEventsFlagValue, numEventsFlagName, DefaultNumEvents, fmt.Sprintf("Number of events to be fetched (default %d)", DefaultNumEvents))
}

func run(cmd *cobra.Command, args []string) error {
	time.Sleep(time.Duration(delayFlagValue) * time.Second)

	logBuffer := bytes.Buffer{}
	log := newSupportArchiveLogger(&logBuffer)
	installconfig.ReadModulesToLogger(log)
	version.LogVersionToLogger(log)

	archiveTargetFile := os.Stdout
	supportArchive := newZipArchive(archiveTargetFile)

	defer archiveTargetFile.Close()
	defer supportArchive.Close()

	err := runCollectors(log, supportArchive)
	if err != nil {
		return err
	}

	// make sure to run this collector at the very end
	return newSupportArchiveOutputCollector(log, supportArchive, &logBuffer).Do()
}

func getAppNameLabel(ctx context.Context, pods clientgocorev1.PodInterface) string {
	podName := os.Getenv(k8senv.PodName)
	if podName != "" {
		options := metav1.GetOptions{}

		pod, err := pods.Get(ctx, podName, options)
		if err != nil {
			return defaultOperatorAppName
		}

		return pod.Labels[k8slabel.AppNameLabel]
	}

	return defaultOperatorAppName
}

func runCollectors(log logd.Logger, supportArchive archiver) error {
	ctx := context.Background()

	kubeConfig, err := config.GetConfig()
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

	logInfof(log, "%s=%s", k8slabel.AppNameLabel, appName)

	fileSize := loadsimFileSizeFlagValue * Mebi
	collectors := []collector{
		newOperatorVersionCollector(log, supportArchive),
		newLogCollector(ctx, log, supportArchive, pods, appName, collectManagedLogsFlagValue),
		newFsLogCollector(ctx, kubeConfig, &remotecommand.DefaultExecutor{}, log, supportArchive, pods, appName, collectManagedLogsFlagValue),
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

func clusterOptions(opts *cluster.Options) {
	opts.Scheme = scheme.Scheme
}
