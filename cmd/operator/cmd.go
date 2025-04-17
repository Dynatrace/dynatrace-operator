package operator

import (
	"context"
	"os"
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	use = "operator"
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

	operatorManager, err := createOperatorManager(kubeCfg, namespace, isOLM)
	if err != nil {
		return err
	}

	err = checkCRDs(operatorManager)
	if err != nil {
		return err
	}

	if !isOLM {
		err = runCertInit(kubeCfg, namespace)
		if err != nil {
			return err
		}
	}

	return errors.WithStack(operatorManager.Start(ctrl.SetupSignalHandler()))
}

func runLocally(kubeCfg *rest.Config) error {
	namespace := os.Getenv(env.PodNamespace)

	err := runCertInit(kubeCfg, namespace)
	if err != nil {
		return err
	}

	operatorManager, err := createOperatorManager(kubeCfg, namespace, false)
	if err != nil {
		return err
	}

	err = checkCRDs(operatorManager)
	if err != nil {
		return err
	}

	return errors.WithStack(operatorManager.Start(ctrl.SetupSignalHandler()))
}

func checkCRDs(operatorManager manager.Manager) error {
	groupKind := schema.GroupKind{
		Group: v1beta4.GroupVersion.Group,
		Kind:  reflect.TypeOf(dynakube.DynaKube{}).Name(),
	}

	_, err := operatorManager.GetRESTMapper().RESTMapping(groupKind, v1beta4.GroupVersion.Version)
	if err != nil {
		log.Info("missing expected CRD version for DynaKube", "version", v1beta4.GroupVersion.Version)

		return err
	}

	groupKind = schema.GroupKind{
		Group: v1alpha2.GroupVersion.Group,
		Kind:  reflect.TypeOf(edgeconnect.EdgeConnect{}).Name(),
	}

	_, err = operatorManager.GetRESTMapper().RESTMapping(groupKind, v1alpha2.GroupVersion.Version)
	if err != nil {
		log.Info("missing expected CRD version for EdgeConnect", "version", v1alpha2.GroupVersion.Version)

		return err
	}

	return nil
}
