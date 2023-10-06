package status

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/util/kubesystem"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetDynakubeStatus(dynakube *dynatracev1beta1.DynaKube, apiReader client.Reader) error {
	uid, err := kubesystem.GetUID(apiReader)
	if err != nil {
		log.Info("could not get cluster ID")
		return err
	}
	dynakube.Status.KubeSystemUUID = string(uid)
	return nil
}
