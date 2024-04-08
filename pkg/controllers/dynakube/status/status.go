package status

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetKubeSystemUUIDInStatus(ctx context.Context, dk *dynatracev1beta1.DynaKube, apiReader client.Reader) error {
	// The UUID of the kube-system namespace should never change
	if dk.Status.KubeSystemUUID != "" {
		return nil
	}

	uid, err := kubesystem.GetUID(ctx, apiReader)
	if err != nil {
		log.Info("could not get cluster ID")

		return err
	}

	dk.Status.KubeSystemUUID = string(uid)

	return nil
}
