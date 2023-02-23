package status

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	DtClient  dtclient.Client
	ApiReader client.Reader
}

func SetDynakubeStatus(dynakube *dynatracev1beta1.DynaKube, opts Options) error {
	apiReader := opts.ApiReader
	dtClient := opts.DtClient

	uid, err := kubesystem.GetUID(apiReader)
	if err != nil {
		log.Info("could not get cluster ID")
		return err
	}

	latestAgentVersionUnixPaas, err := dtClient.GetLatestAgentVersion(
		dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		log.Info("could not get agent paas unix version")
		return err
	}

	dynakube.Status.KubeSystemUUID = string(uid)
	dynakube.Status.LatestAgentVersionUnixPaas = latestAgentVersionUnixPaas

	return nil
}
