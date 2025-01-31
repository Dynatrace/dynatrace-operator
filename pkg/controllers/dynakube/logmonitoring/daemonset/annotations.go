package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const annotationTenantTokenHash = api.InternalFlagPrefix + "tenant-token-hash"

func (r *Reconciler) buildAnnotations(dk *dynakube.DynaKube) map[string]string {
	annotations := map[string]string{
		annotationTenantTokenHash: dk.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo.TenantTokenHash,
	}

	return annotations
}
