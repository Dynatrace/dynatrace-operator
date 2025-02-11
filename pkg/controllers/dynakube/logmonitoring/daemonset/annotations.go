package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
)

const annotationTenantTokenHash = api.InternalFlagPrefix + "tenant-token-hash"

func (r *Reconciler) getAnnotations() map[string]string {
	return maputils.MergeMap(
		r.dk.LogMonitoring().Template().Annotations,
		r.buildTenantTokenHashAnnotation())
}

func (r *Reconciler) buildTenantTokenHashAnnotation() map[string]string {
	annotations := map[string]string{
		annotationTenantTokenHash: r.dk.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo.TenantTokenHash,
	}

	return annotations
}
