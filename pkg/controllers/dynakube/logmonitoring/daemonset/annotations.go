package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/configsecret"
)

const annotationTenantTokenHash = api.InternalFlagPrefix + "tenant-token-hash"

func (r *Reconciler) getAnnotations() map[string]string {
	return configsecret.AddAnnotations(r.dk.LogMonitoring().Template().Annotations, *r.dk)
}
