package troubleshoot

import (
	"context"
	"net/http"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/dockerkeychain"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		RunE: run,
	}

	addFlags(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&dynakubeFlagValue, dynakubeFlagName, dynakubeFlagShorthand, "", "Specify a different Dynakube name.")
	cmd.PersistentFlags().StringVarP(&namespaceFlagValue, namespaceFlagName, namespaceFlagShorthand, k8senv.DefaultNamespace(), "Specify a different Namespace.")
}

func clusterOptions(opts *cluster.Options) {
	opts.Scheme = scheme.Scheme
}

func run(cmd *cobra.Command, args []string) error {
	version.LogVersion()
	logd.LogBaseLoggerSettings()

	kubeConfig, err := config.GetConfig()
	if err != nil {
		return err
	}

	log := NewTroubleshootLoggerToWriter(os.Stdout)

	RunTroubleshootCmd(cmd.Context(), log, namespaceFlagValue, kubeConfig)

	return nil
}

func RunTroubleshootCmd(ctx context.Context, log logd.Logger, namespaceName string, kubeConfig *rest.Config) {
	apiReader, err := GetK8SClusterAPIReader(kubeConfig)
	if err != nil {
		return
	}

	err = checkNamespace(ctx, log, apiReader, namespaceName)
	if err != nil {
		logErrorf(log, "prerequisite checks failed, aborting (%v)", err)

		return
	}

	dks, err := getDynakubes(ctx, log, apiReader, namespaceName, dynakubeFlagValue)
	if checkCRD(log, err) != nil {
		logErrorf(log, "error during getting dynakubes: %v", err)

		return
	}

	runChecksForAllDynakubes(ctx, log, apiReader, &http.Client{}, dks)
}

func GetK8SClusterAPIReader(kubeConfig *rest.Config) (client.Reader, error) {
	k8scluster, err := cluster.New(kubeConfig, clusterOptions)
	if err != nil {
		return nil, err
	}

	return k8scluster.GetAPIReader(), nil
}

func runChecksForAllDynakubes(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, httpClient *http.Client, dynakubes []dynakube.DynaKube) {
	for _, dk := range dynakubes {
		err := runChecksForDynakube(ctx, baseLog, apiReader, httpClient, dk)
		if err != nil {
			logErrorf(baseLog, "Error in DynaKube %s/%s", dk.Namespace, dk.Name)
		}
	}
}

func runChecksForDynakube(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, httpClient *http.Client, dk dynakube.DynaKube) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logNewCheckf(log, "checking if '%s:%s' Dynakube is configured correctly", dk.Namespace, dk.Name)
	logInfof(log, "using '%s:%s' Dynakube", dk.Namespace, dk.Name)

	pullSecret, err := checkDynakube(ctx, baseLog, apiReader, &dk)
	if err != nil {
		return errors.Wrapf(err, "'%s:%s' Dynakube isn't valid. %s",
			dk.Namespace, dk.Name, dynakubeNotValidMessage())
	}

	logOkf(log, "'%s:%s' Dynakube is valid", dk.Namespace, dk.Name)

	keychain, err := dockerkeychain.NewDockerKeychain(ctx, apiReader, pullSecret)
	if err != nil {
		return err
	}

	transport, err := createTransport(ctx, apiReader, &dk, httpClient)
	if err != nil {
		return err
	}

	err = verifyAllImagesAvailable(ctx, log, keychain, transport, &dk)
	if err != nil {
		return err
	}

	return checkProxySettings(ctx, log, apiReader, &dk)
}

func createTransport(ctx context.Context, apiReader client.Reader, dk *dynakube.DynaKube, httpClient *http.Client) (*http.Transport, error) {
	var transport *http.Transport
	if httpClient != nil && httpClient.Transport != nil {
		transport = httpClient.Transport.(*http.Transport).Clone()
	} else {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}

	return registry.PrepareTransportForDynaKube(ctx, apiReader, transport, dk)
}

func getDynakubes(ctx context.Context, log logd.Logger, apiReader client.Reader, namespaceName string, dynakubeName string) ([]dynakube.DynaKube, error) {
	var err error

	var dynakubes []dynakube.DynaKube

	if dynakubeName == "" {
		logNewDynakubef(log, "no Dynakube specified - checking all Dynakubes in namespace '%s'", namespaceName)

		dynakubes, err = getAllDynakubesInNamespace(ctx, log, apiReader, namespaceName)
		if err != nil {
			return nil, err
		}
	} else {
		dk, err := getSelectedDynakube(ctx, apiReader, namespaceName, dynakubeName)
		if err != nil {
			return nil, err
		}

		dynakubes = append(dynakubes, dk)
	}

	return dynakubes, nil
}

func getAllDynakubesInNamespace(ctx context.Context, log logd.Logger, apiReader client.Reader, namespaceName string) ([]dynakube.DynaKube, error) {
	var dynakubes dynakube.DynaKubeList

	err := apiReader.List(ctx, &dynakubes, client.InNamespace(namespaceName))
	if err != nil {
		logErrorf(log, "failed to list Dynakubes: %v", err)

		return nil, err
	}

	if len(dynakubes.Items) == 0 {
		logErrorf(log, "no Dynakubes found in namespace '%s'", namespaceName)

		return nil, nil
	}

	return dynakubes.Items, nil
}
