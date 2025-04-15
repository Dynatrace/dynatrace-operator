package operator

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	use             = "operator"
	expectedVersion = "v1beta4"
)

func New() *cobra.Command {
	return &cobra.Command{
		Use:          use,
		RunE:         run(),
		SilenceUsage: true,
	}
}

func run() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		installconfig.ReadModules()
		version.LogVersion()
		logd.LogBaseLoggerSettings()

		kubeCfg, err := config.GetConfig()
		if err != nil {
			return err
		}

		if kubesystem.IsRunLocally() {
			log.Info("running locally in debug mode")

			return runLocally(kubeCfg)
		}

		return runInPod(kubeCfg)
	}
}

func runInPod(kubeCfg *rest.Config) error {
	clt, err := client.New(kubeCfg, client.Options{})
	if err != nil {
		return err
	}

	podName := os.Getenv(env.PodName)
	namespace := os.Getenv(env.PodNamespace)

	operatorPod, err := pod.Get(context.Background(), clt, podName, namespace)
	if err != nil {
		return err
	}

	isOLM := kubesystem.IsDeployedViaOlm(*operatorPod)
	if !isOLM {
		err = runCertInit(kubeCfg, namespace)
		if err != nil {
			return err
		}
	}

	return runOperator(kubeCfg, namespace, isOLM)
}

func runLocally(kubeCfg *rest.Config) error {
	namespace := os.Getenv(env.PodNamespace)

	err := runCertInit(kubeCfg, namespace)
	if err != nil {
		return err
	}

	return runOperator(kubeCfg, namespace, false)
}

func runOperator(kubeCfg *rest.Config, namespace string, isOLM bool) error {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(kubeCfg)
	if err != nil {
		return err
	}

	groupVersion := schema.GroupVersion{
		Group:   "dynatrace.com",
		Version: expectedVersion,
	}

	resourceList, err := discoveryClient.ServerResourcesForGroupVersion(groupVersion.String())
	if err != nil {
		return errors.WithMessagef(err, "required DynaKube CRD version %s not found", groupVersion.String())
	}

	var versionFound bool

	for _, resource := range resourceList.APIResources {
		if resource.Kind == "DynaKube" {
			versionFound = true

			break
		}
	}

	if !versionFound {
		return errors.New("DynaKube kind not found in group version")
	}

	operatorManager, err := createOperatorManager(kubeCfg, namespace, isOLM)
	if err != nil {
		return err
	}

	ctx := ctrl.SetupSignalHandler()
	err = operatorManager.Start(ctx)

	return errors.WithStack(err)
}
