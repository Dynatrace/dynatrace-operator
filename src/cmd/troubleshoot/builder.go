package troubleshoot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	dynakubeversion "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
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

		log := NewTroubleshootLoggerToWriter(os.Stdout)

		RunTroubleshootCmd(cmd.Context(), log, namespaceFlagValue, kubeConfig)
		return nil
	}
}
func RunTroubleshootCmd(ctx context.Context, log logr.Logger, namespaceName string, kubeConfig *rest.Config) {
	err := checkOneAgentAPM(log, kubeConfig)
	if err != nil {
		logErrorf(log, "prerequisite checks failed, aborting (%v)", err)
		return
	}
	apiReader, err := GetK8SClusterAPIReader(kubeConfig)
	if err != nil {
		return
	}
	err = checkNamespace(ctx, log, apiReader, namespaceName)
	if err != nil {
		logErrorf(log, "prerequisite checks failed, aborting (%v)", err)
		return
	}
	dynakubes := &dynatracev1beta1.DynaKubeList{}
	err = apiReader.List(ctx, dynakubes, &client.ListOptions{Namespace: namespaceName})

	if err != nil {
		err := DetermineDynakubeError(err)
		logErrorf(log, "prerequisite checks failed, aborting (%v)", err)
		return
	}
	logOkf(log, "CRD for Dynakube exists")

	runChecksForAllDynakubes(ctx, log, apiReader, &http.Client{}, dynakubes.Items)
}

func GetK8SClusterAPIReader(kubeConfig *rest.Config) (client.Reader, error) {
	k8scluster, err := cluster.New(kubeConfig, clusterOptions)
	if err != nil {
		return nil, err
	}
	return k8scluster.GetAPIReader(), nil
}

func runChecksForAllDynakubes(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, httpClient *http.Client, dynakubes []dynatracev1beta1.DynaKube) {
	for _, dynakube := range dynakubes {
		err := runChecksForDynakube(ctx, baseLog, apiReader, httpClient, dynakube)
		if err != nil {
			logErrorf(baseLog, "Error in DynaKube %s/%s", dynakube.Namespace, dynakube.Name)
		}
	}
}

func runChecksForDynakube(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, httpClient *http.Client, dynakube dynatracev1beta1.DynaKube) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logNewCheckf(log, "checking if '%s:%s' Dynakube is configured correctly", dynakube.Namespace, dynakube.Name)
	logInfof(baseLog, "using '%s:%s' Dynakube", dynakube.Namespace, dynakube.Name)

	pullSecret, err := checkDynakube(ctx, baseLog, apiReader, &dynakube)
	if err != nil {
		return errors.Wrapf(err, "'%s:%s' Dynakube isn't valid. %s",
			dynakube.Namespace, dynakube.Name, dynakubeNotValidMessage())
	}
	logOkf(log, "'%s:%s' Dynakube is valid", dynakube.Namespace, dynakube.Name)

	keychain, err := dockerkeychain.NewDockerKeychain(ctx, apiReader, pullSecret)
	if err != nil {
		return err
	}

	transport, err := createTransport(ctx, apiReader, &dynakube, log, httpClient)
	if err != nil {
		return err
	}

	err = verifyAllImagesAvailable(ctx, baseLog, keychain, transport, &dynakube)
	if err != nil {
		return err
	}
	err = checkProxySettingsWithLog(ctx, baseLog, apiReader, &dynakube)
	if err != nil {
		return err
	}
	return nil
}

func createTransport(ctx context.Context, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube, log logr.Logger, httpClient *http.Client) (*http.Transport, error) {
	proxy, err := getProxyURL(ctx, apiReader, dynakube)
	if err != nil {
		return nil, err
	}
	err = applyProxy(log, httpClient, proxy)
	if err != nil {
		return nil, err
	}
	var transport *http.Transport
	if httpClient != nil && httpClient.Transport != nil {
		transport = httpClient.Transport.(*http.Transport).Clone()
	} else {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}

	transport, err = dynakubeversion.PrepareTransport(ctx, apiReader, transport, dynakube)
	if err != nil {
		return nil, err
	}
	return transport, nil
}

func applyProxy(log logr.Logger, httpClient *http.Client, proxy string) error {
	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return errors.Wrap(err, "could not parse proxy URL!")
		}

		if httpClient.Transport == nil {
			httpClient.Transport = http.DefaultTransport
		}

		httpClient.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyUrl)
		logInfof(log, "using '%s' proxy to connect to the registry", proxyUrl.Host)
	}

	return nil
}

func getDynakubes(ctx context.Context, log logr.Logger, apiReader client.Reader, namespaceName string, dynakubeName string) ([]dynatracev1beta1.DynaKube, error) {
	var err error
	var dynakubes []dynatracev1beta1.DynaKube

	if dynakubeName == "" {
		logNewDynakubef(log, "no Dynakube specified - checking all Dynakubes in namespace '%s'", namespaceName)
		dynakubes, err = getAllDynakubesInNamespace(ctx, log, apiReader, namespaceName)
		if err != nil {
			return nil, err
		}
	} else {
		dynakube, err := getSelectedDynakube(ctx, apiReader, namespaceName, dynakubeName)
		if err != nil {
			return nil, err
		}
		dynakubes = append(dynakubes, dynakube)
	}

	return dynakubes, nil
}

func getAllDynakubesInNamespace(ctx context.Context, log logr.Logger, apiReader client.Reader, namespaceName string) ([]dynatracev1beta1.DynaKube, error) {
	var dynakubes dynatracev1beta1.DynaKubeList
	err := apiReader.List(ctx, &dynakubes, client.InNamespace(namespaceName))

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
