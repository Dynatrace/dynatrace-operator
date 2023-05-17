package status

import (
	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetDynakubeStatus(dynakube *dynatracev1.DynaKube, apiReader client.Reader) error {
	uid, err := kubesystem.GetUID(apiReader)
	if err != nil {
		log.Info("could not get cluster ID")
		return err
	}
	dynakube.Status.KubeSystemUUID = string(uid)
	return nil
}
