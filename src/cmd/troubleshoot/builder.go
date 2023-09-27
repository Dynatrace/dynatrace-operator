package troubleshoot

import (
	"context"
	"fmt"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
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

		k8scluster, err := builder.GetCluster(kubeConfig)
		if err != nil {
			return err
		}

		apiReader := k8scluster.GetAPIReader()

		log := NewTroubleshootLoggerToWriter(os.Stdout)

		RunTroubleshootCmd(cmd.Context(), log, apiReader, namespaceFlagValue, kubeConfig)
		return nil
	}
}

func RunTroubleshootCmd(ctx context.Context, log logr.Logger, apiReader client.Reader, namespaceName string, kubeConfig *rest.Config) {
	err := runPrerequisiteChecks(ctx, log, apiReader, namespaceName, kubeConfig) // ignore error to avoid polluting pretty logs

	if err != nil {
		logErrorf(log, "prerequisite checks failed, aborting")
		return
	}

	dynakubes, err := getDynakubes(ctx, log, apiReader, namespaceName, dynakubeFlagValue)
	if err != nil {
		logErrorf(log, "error reading Dynakubes")
		return
	}
	runChecksForAllDynakubes(ctx, log, apiReader, namespaceName, dynakubes)
}

func runChecksForAllDynakubes(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, namespaceName string, dynakubes []dynatracev1beta1.DynaKube) {
	for _, dynakube := range dynakubes {
		dynakube, err := getSelectedDynakube(ctx, apiReader, namespaceName, dynakube.Name)
		if err != nil {
			logErrorf(baseLog, "Could not get DynaKube %s/%s", dynakube.Namespace, dynakube.Name)
		}

		err = runChecksForDynakube(ctx, baseLog, apiReader, namespaceName, &dynakube)
		if err != nil {
			logErrorf(baseLog, "Error in DynaKube %s/%s", dynakube.Namespace, dynakube.Name)
		}
	}
}

func runChecksForDynakube(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, namespaceName string, dynakube *dynatracev1beta1.DynaKube) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logNewCheckf(log, "checking if '%s:%s' Dynakube is configured correctly", namespaceName, dynakube.Name)
	logInfof(baseLog, "using '%s:%s' Dynakube", namespaceName, dynakube.Name)

	err := checkDynakube(ctx, baseLog, apiReader, namespaceName, dynakube)
	if err != nil {
		return errors.Wrapf(err, "'%s:%s' Dynakube isn't valid. %s",
			namespaceName, dynakube.Name, dynakubeNotValidMessage())
	}
	logOkf(log, "'%s:%s' Dynakube is valid", namespaceName, dynakube.Name)
	// TODO: verifyAllImagesAvailable
	// TODO: checkProxySettings
	return nil
}

func runPrerequisiteChecks(ctx context.Context, log logr.Logger, apiReader client.Reader, namespaceName string, kubeConfig *rest.Config) error {
	err := checkNamespace(ctx, log, apiReader, namespaceName)
	if err != nil {
		return err
	}
	err = checkCRD(ctx, log, apiReader, namespaceName)
	if err != nil {
		return err
	}
	err = checkOneAgentAPM(log, kubeConfig)
	if err != nil {
		return err
	}
	return nil
}

func getDynakubes(
	ctx context.Context,
	log logr.Logger,
	reader client.Reader,
	namespaceName string,
	dynakubeName string,
) ([]dynatracev1beta1.DynaKube, error) {
	var err error
	var dynakubes []dynatracev1beta1.DynaKube

	if dynakubeName == "" {
		logNewDynakubef(log, "no Dynakube specified - checking all Dynakubes in namespace '%s'", namespaceName)
		dynakubes, err = getAllDynakubesInNamespace(ctx, log, reader, namespaceName)
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

func getAllDynakubesInNamespace(ctx context.Context, log logr.Logger, reader client.Reader, namespaceName string) ([]dynatracev1beta1.DynaKube, error) {
	var dynakubes dynatracev1beta1.DynaKubeList
	err := reader.List(ctx, &dynakubes, client.InNamespace(namespaceName))

	if err != nil {
		logErrorf(log, "failed to list Dynakubes: %v", err)
		return nil, err
	}

	if len(dynakubes.Items) == 0 {
		err = fmt.Errorf("no Dynakubes found in namespace '%s'", namespaceName)
		logErrorf(log, err.Error())
		return nil, err
	}

	return dynakubes.Items, nil
}
