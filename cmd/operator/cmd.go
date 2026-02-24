package operator

import (
	"context"
	"os"
	"reflect"

	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest" //nolint:revive
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/envvars"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/system"
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
		RunE:         run,
		SilenceUsage: true,
	}
}

func run(cmd *cobra.Command, args []string) error {
	installconfig.ReadModules()
	version.LogVersion()
	logd.LogBaseLoggerSettings()

	kubeCfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	if system.IsRunLocally() {
		log.Info("running locally in debug mode")

		return runLocally(kubeCfg)
	}

	return runInPod(kubeCfg)
}

func runInPod(kubeCfg *rest.Config) error {
	clt, err := client.New(kubeCfg, client.Options{})
	if err != nil {
		return err
	}

	podName := os.Getenv(k8senv.PodName)
	namespace := os.Getenv(k8senv.PodNamespace)

	operatorPod, err := k8spod.Get(context.Background(), clt, podName, namespace)
	if err != nil {
		return err
	}

	isOLM := system.IsDeployedViaOlm(*operatorPod)

	if !isOLM {
		err = runCertInit(kubeCfg, namespace)
		if err != nil {
			return err
		}
	}

	if shouldRunCRDStorageMigrationInitManager() {
		err = runCRDStorageMigration(kubeCfg, namespace)
		if err != nil {
			return err
		}
	}

	operatorManager, err := createOperatorManager(kubeCfg, namespace, isOLM)
	if err != nil {
		return err
	}

	if isOLM {
		// in most cases checkCRDs happen in the runCertInit,
		// the reason for that is we run a manager to create the certs
		// this manager uses the same ports for livez
		// the controller-runtime will error with "port already in use" even if we didn't .Start the manager
		err = checkCRDs(operatorManager)
		if err != nil {
			return err
		}
	}

	return errors.WithStack(operatorManager.Start(ctrl.SetupSignalHandler()))
}

func runLocally(kubeCfg *rest.Config) error {
	namespace := os.Getenv(k8senv.PodNamespace)

	err := runCertInit(kubeCfg, namespace)
	if err != nil {
		return err
	}

	if shouldRunCRDStorageMigrationInitManager() {
		err = runCRDStorageMigration(kubeCfg, namespace)
		if err != nil {
			return err
		}
	}

	operatorManager, err := createOperatorManager(kubeCfg, namespace, false)
	if err != nil {
		return err
	}

	return errors.WithStack(operatorManager.Start(ctrl.SetupSignalHandler()))
}

func shouldRunCRDStorageMigrationInitManager() bool {
	return envvars.GetBool(consts.CRDStorageMigrationEnvVar, true)
}

func checkCRDs(operatorManager manager.Manager) error {
	groupKind := schema.GroupKind{
		Group: latest.GroupVersion.Group,
		Kind:  reflect.TypeFor[dynakube.DynaKube]().Name(),
	}

	_, err := operatorManager.GetRESTMapper().RESTMapping(groupKind, latest.GroupVersion.Version)
	if err != nil {
		log.Info("missing expected CRD version for DynaKube", "version", latest.GroupVersion.Version)

		return err
	}

	if installconfig.GetModules().EdgeConnect {
		groupKind = schema.GroupKind{
			Group: v1alpha2.GroupVersion.Group,
			Kind:  reflect.TypeFor[edgeconnect.EdgeConnect]().Name(),
		}

		_, err = operatorManager.GetRESTMapper().RESTMapping(groupKind, v1alpha2.GroupVersion.Version)
		if err != nil {
			log.Info("missing expected CRD version for EdgeConnect", "version", v1alpha2.GroupVersion.Version)

			return err
		}
	}

	return nil
}
