package troubleshoot

import (
	"context"
	"net/http"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

const (
	use                    = "troubleshoot"
	dynakubeFlagName       = "dynakube"
	dynakubeFlagShorthand  = "d"
	namespaceFlagName      = "namespace"
	namespaceFlagShorthand = "n"
)

var (
	dynakubeFlagValue  string
	namespaceFlagValue string
)

type CommandBuilder struct {
	configProvider config.Provider
}

func NewTroubleshootCommandBuilder() CommandBuilder {
	return CommandBuilder{}
}

func (builder CommandBuilder) SetConfigProvider(provider config.Provider) CommandBuilder {
	builder.configProvider = provider
	return builder
}

func (builder CommandBuilder) GetCluster(kubeConfig *rest.Config) (cluster.Cluster, error) {
	return cluster.New(kubeConfig, clusterOptions)
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
	cmd.PersistentFlags().StringVarP(&dynakubeFlagValue, dynakubeFlagName, dynakubeFlagShorthand, "", "Specify a different Dynakube name.")
	cmd.PersistentFlags().StringVarP(&namespaceFlagValue, namespaceFlagName, namespaceFlagShorthand, defaultNamespace(), "Specify a different Namespace.")
}

func defaultNamespace() string {
	namespace := os.Getenv("POD_NAMESPACE")

	if namespace == "" {
		return "dynatrace"
	}
	return namespace
}

func clusterOptions(opts *cluster.Options) {
	opts.Scheme = scheme.Scheme
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		kubeConfig, err := builder.configProvider.GetConfig()
		if err != nil {
			return err
		}

		err = dynatracev1beta1.AddToScheme(scheme.Scheme)
		if err != nil {
			return err
		}

		k8scluster, err := builder.GetCluster(kubeConfig)
		if err != nil {
			return err
		}

		apiReader := k8scluster.GetAPIReader()
		dynakubeUnrelatedTests := []troubleshootFunc{
			checkNamespace,
		}

		troubleshootCtx := troubleshootContext{
			context:       context.Background(),
			apiReader:     apiReader,
			httpClient:    &http.Client{},
			namespaceName: namespaceFlagValue,
		}
		for _, test := range dynakubeUnrelatedTests {
			err = test(&troubleshootCtx)
			if err != nil {
				logErrorf(err.Error())
				return nil
			}
		}
		resetLog()

		perDynakubeTests := []troubleshootFunc{
			checkDynakube,
			checkDTClusterConnection,
			checkImagePullable,
		}

		dynakubes, err := getDynakubes(&troubleshootCtx)
		if err != nil {
			return nil
		}

		for _, dynakube := range dynakubes {
			troubleshootCtx = troubleshootContext{
				context:       context.Background(),
				apiReader:     apiReader,
				httpClient:    &http.Client{},
				namespaceName: namespaceFlagValue,
				dynakube:      dynakube,
				dynakubeName:  dynakube.Name,
			}
			logNewDynakubef(troubleshootCtx.dynakubeName)
			for _, test := range perDynakubeTests {
				err = test(&troubleshootCtx)
				if err != nil {
					logErrorf(err.Error())
					return nil
				}
			}
			resetLog()
			logOkf("'%s' - all checks passed", troubleshootCtx.dynakubeName)
		}
		return nil
	}
}

func getDynakubes(troubleshootCtx *troubleshootContext) ([]dynatracev1beta1.DynaKube, error) {
	var err error
	var dynakubes []dynatracev1beta1.DynaKube
	if dynakubeFlagValue == "" {
		logInfof("no Dynakube specified - checking all Dynakubes in namespace '%s'", namespaceFlagValue)
		dynakubes, err = getAllDynakubesInNamespace(troubleshootCtx)
		if err != nil {
			return nil, err
		}
	} else {
		dynakube := dynatracev1beta1.DynaKube{}
		dynakube.Name = dynakubeFlagValue
		dynakubes = append(dynakubes, dynakube)
	}
	return dynakubes, nil
}

func getAllDynakubesInNamespace(troubleshootContext *troubleshootContext) ([]dynatracev1beta1.DynaKube, error) {
	query := kubeobjects.NewDynakubeQuery(troubleshootContext.apiReader, troubleshootContext.namespaceName).WithContext(troubleshootContext.context)
	dynakubes, err := query.List()
	if err != nil {
		logErrorf("failed to list Dynakubes: %v", err)
		return nil, err
	}
	if len(dynakubes.Items) == 0 {
		logErrorf("no Dynakubes found in namespace %s", namespaceFlagValue)
		return nil, err
	}
	return dynakubes.Items, nil
}
