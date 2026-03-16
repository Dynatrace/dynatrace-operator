package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/configsecret"
)

const annotationTenantTokenHash = api.InternalFlagPrefix + "tenant-token-hash"

func (r *Reconciler) getAnnotations(dk *dynakube.DynaKube) map[string]string {
	return configsecret.AddAnnotations(dk.LogMonitoring().Template().Annotations, *dk)
}
