package troubleshoot

import (
	"context"
	"fmt"
	"net/http"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

const (
	use                    = "troubleshoot"
	dynakubeFlagName       = "dynakube"
	dynakubeFlagShorthand  = "d"
	namespaceFlagName      = "namespace"
	namespaceFlagShorthand = "n"

	namespaceCheckName           = "namespace"
	crdCheckName                 = "crd"
	dynakubeCheckName            = "dynakube"
	oneAgentAPMCheckName         = "oneAgentAPM"
	dtClusterConnectionCheckName = "DynatraceClusterConnection"
	imagePullableCheckName       = "imagePullable"
	proxySettingsCheckName       = "proxySettings"
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
	cmd.PersistentFlags().StringVarP(&namespaceFlagValue, namespaceFlagName, namespaceFlagShorthand, kubeobjects.DefaultNamespace(), "Specify a different Namespace.")
}

func clusterOptions(opts *cluster.Options) {
	opts.Scheme = scheme.Scheme
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		version.LogVersion()

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

		troubleshootCtx := troubleshootContext{
			context:       context.Background(),
			apiReader:     apiReader,
			httpClient:    &http.Client{},
			namespaceName: namespaceFlagValue,
			kubeConfig:    *kubeConfig,
		}

		results := NewChecksResults()
		err = runChecks(results, &troubleshootCtx, getPrerequisiteChecks()) // ignore error to avoid polluting pretty logs
		resetLogger()
		if err != nil {
			logErrorf("prerequisite checks failed, aborting")
			return nil //nolint:nilerr
		}

		dynakubes, err := getDynakubes(troubleshootCtx, dynakubeFlagValue)
		if err != nil {
			return nil //nolint:nilerr
		}

		runChecksForAllDynakubes(results, getDynakubeSpecificChecks(results), dynakubes, apiReader)

		return nil
	}
}

func runChecksForAllDynakubes(results ChecksResults, checks []*Check, dynakubes []dynatracev1beta1.DynaKube, apiReader client.Reader) {
	for _, dynakube := range dynakubes {
		results.checkResultMap = map[*Check]Result{}
		logNewDynakubef(dynakube.Name)

		troubleshootCtx := troubleshootContext{
			context:       context.Background(),
			apiReader:     apiReader,
			httpClient:    &http.Client{},
			namespaceName: namespaceFlagValue,
			dynakube:      dynakube,
		}

		_ = runChecks(results, &troubleshootCtx, checks) // ignore error to avoid polluting pretty logs
		resetLogger()
		if !results.hasErrors() {
			logOkf("'%s' - all checks passed", dynakube.Name)
		}
	}
}

func getPrerequisiteChecks() []*Check {
	namespaceCheck := &Check{
		Name: namespaceCheckName,
		Do:   checkNamespace,
	}
	crdCheck := &Check{
		Name: crdCheckName,
		Do:   checkCRD,
	}
	oneAgentAPMCheck := &Check{
		Name: oneAgentAPMCheckName,
		Do:   checkOneAgentAPM,
	}
	return []*Check{namespaceCheck, crdCheck, oneAgentAPMCheck}
}

func getDynakubeSpecificChecks(results ChecksResults) []*Check {
	dynakubeCheck := &Check{
		Name: dynakubeCheckName,
		Do: func(troubleshootCtx *troubleshootContext) error {
			return checkDynakube(results, troubleshootCtx)
		},
	}
	imagePullableCheck := &Check{
		Name:          imagePullableCheckName,
		Do:            verifyAllImagesAvailable,
		Prerequisites: []*Check{dynakubeCheck},
	}
	proxySettingsCheck := &Check{
		Name:          proxySettingsCheckName,
		Do:            checkProxySettings,
		Prerequisites: []*Check{dynakubeCheck},
	}
	return []*Check{dynakubeCheck, imagePullableCheck, proxySettingsCheck}
}

func getDynakubes(troubleshootCtx troubleshootContext, dynakubeName string) ([]dynatracev1beta1.DynaKube, error) {
	var err error
	var dynakubes []dynatracev1beta1.DynaKube

	if dynakubeName == "" {
		logNewDynakubef("no Dynakube specified - checking all Dynakubes in namespace '%s'", troubleshootCtx.namespaceName)
		dynakubes, err = getAllDynakubesInNamespace(troubleshootCtx)
		if err != nil {
			return nil, err
		}
	} else {
		dynakube := dynatracev1beta1.DynaKube{}
		dynakube.Name = dynakubeName
		dynakubes = append(dynakubes, dynakube)
	}

	return dynakubes, nil
}

func getAllDynakubesInNamespace(troubleshootContext troubleshootContext) ([]dynatracev1beta1.DynaKube, error) {
	var dynakubes dynatracev1beta1.DynaKubeList
	err := troubleshootContext.apiReader.List(troubleshootContext.context, &dynakubes, client.InNamespace(troubleshootContext.namespaceName))

	if err != nil {
		logErrorf("failed to list Dynakubes: %v", err)
		return nil, err
	}

	if len(dynakubes.Items) == 0 {
		err = fmt.Errorf("no Dynakubes found in namespace '%s'", troubleshootContext.namespaceName)
		logErrorf(err.Error())
		return nil, err
	}

	return dynakubes.Items, nil
}
